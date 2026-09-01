package handler

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/friend-diaries/backend/internal/infrastructure/repository"
	"github.com/friend-diaries/backend/internal/infrastructure/storage"
	"github.com/friend-diaries/backend/internal/interfaces/http/middleware"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	maxImageSize     = 10 << 20  // 10 MB
	maxVideoSize     = 32 << 20  // 32 MB after client-side compression
	maxVideoDuration = 180.5     // Small allowance for container rounding.
	maxVideoBitrate  = 1_600_000 // Average bits/second, including audio.
	presignedExpiry  = 24 * time.Hour
)

var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
	"image/heic": ".heic",
}

type MediaHandler struct {
	storage   *storage.Client
	imageRepo *repository.NoteImageRepository
	userRepo  user.Repository
}

func NewMediaHandler(storage *storage.Client, imageRepo *repository.NoteImageRepository, userRepo user.Repository) *MediaHandler {
	return &MediaHandler{storage: storage, imageRepo: imageRepo, userRepo: userRepo}
}

type UploadResponse struct {
	ID              string   `json:"id"`
	MediaURL        string   `json:"media_url"`
	MediaKind       string   `json:"media_kind"`
	MimeType        string   `json:"mime_type"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
}

// Upload handles multipart image upload, stores in MinIO, and returns the image ID.
// The image is initially unattached (note_id = NULL) and gets linked when the note is created.
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxVideoSize+(1<<20))
	if err := r.ParseMultipartForm(maxVideoSize); err != nil {
		response.Error(w, apperror.BadRequest("media upload is too large"))
		return
	}

	file, header, err := r.FormFile("media")
	if err != nil {
		// Backwards compatibility with older clients.
		file, header, err = r.FormFile("image")
	}
	if err != nil {
		response.Error(w, apperror.BadRequest("missing media field"))
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	// Normalise e.g. "image/jpeg; charset=..." → "image/jpeg"
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	if strings.HasPrefix(contentType, "video/") {
		h.uploadVideo(w, r, file, header.Size, contentType, userID)
		return
	}

	ext, ok := allowedMimeTypes[contentType]
	if !ok {
		response.Error(w, apperror.BadRequest(fmt.Sprintf("unsupported media type: %s", contentType)))
		return
	}
	if header.Size > maxImageSize {
		response.Error(w, apperror.BadRequest("image must be 10 MB or smaller"))
		return
	}

	slog.Info("media upload: processing image", "content_type", contentType, "size", header.Size)

	// Resize and re-encode to WebP regardless of input format
	imageData, width, height, procErr := storage.ProcessImage(file, contentType)
	if procErr != nil {
		slog.Error("media upload: image processing failed", "content_type", contentType, "error", procErr)
		response.Error(w, apperror.BadRequest("could not process image: "+procErr.Error()))
		return
	}
	_ = ext // no longer needed — always .webp
	slog.Info("media upload: image processed", "width", width, "height", height, "webp_size", len(imageData))

	objectKey := fmt.Sprintf("images/%s/%s.webp", userID, uuid.New())
	webpSize := int64(len(imageData))

	slog.Info("media upload: uploading to storage", "key", objectKey)
	if err := h.storage.Upload(r.Context(), objectKey, bytes.NewReader(imageData), webpSize, "image/webp"); err != nil {
		slog.Error("media upload: storage upload failed", "key", objectKey, "error", err)
		response.Error(w, apperror.Internal("uploading image", err))
		return
	}
	slog.Info("media upload: stored in minio", "key", objectKey)

	img := &repository.NoteImage{
		UploaderID: userID,
		ObjectKey:  objectKey,
		MimeType:   "image/webp",
		MediaKind:  "image",
		SizeBytes:  webpSize,
	}
	if err := h.imageRepo.Create(r.Context(), img); err != nil {
		slog.Error("media upload: db insert failed", "key", objectKey, "error", err)
		// best-effort cleanup
		_ = h.storage.Delete(r.Context(), objectKey)
		response.Error(w, apperror.Internal("saving image record", err))
		return
	}

	response.Created(w, UploadResponse{
		ID:        img.ID.String(),
		MediaURL:  fmt.Sprintf("/api/media/%s", img.ID),
		MediaKind: "image",
		MimeType:  "image/webp",
	})
}

func (h *MediaHandler) uploadVideo(
	w http.ResponseWriter,
	r *http.Request,
	file io.Reader,
	declaredSize int64,
	contentType string,
	userID uuid.UUID,
) {
	currentUser, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil || !currentUser.IsPremium {
		response.Error(w, apperror.Forbidden("video memories are available on Premium"))
		return
	}
	if contentType != "video/webm" && contentType != "video/mp4" {
		response.Error(w, apperror.BadRequest("videos must be compressed as WebM or MP4"))
		return
	}
	if declaredSize <= 0 || declaredSize > maxVideoSize {
		response.Error(w, apperror.BadRequest("compressed video must be 32 MB or smaller"))
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxVideoSize+1))
	if err != nil || len(data) == 0 || len(data) > maxVideoSize {
		response.Error(w, apperror.BadRequest("compressed video must be 32 MB or smaller"))
		return
	}
	duration, err := storage.ProbeVideoDuration(data, contentType)
	if err != nil {
		response.Error(w, apperror.BadRequest("video container is invalid or missing duration metadata"))
		return
	}
	if duration > maxVideoDuration {
		response.Error(w, apperror.BadRequest("videos can be at most 3 minutes long"))
		return
	}
	averageBitrate := float64(len(data)*8) / duration
	if averageBitrate > maxVideoBitrate {
		response.Error(w, apperror.BadRequest("video is not compressed enough; use the in-app compressor"))
		return
	}

	ext := filepath.Ext(strings.TrimSpace(r.FormValue("filename")))
	if contentType == "video/webm" {
		ext = ".webm"
	} else {
		ext = ".mp4"
	}
	objectKey := fmt.Sprintf("videos/%s/%s%s", userID, uuid.New(), ext)
	if err := h.storage.Upload(r.Context(), objectKey, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		response.Error(w, apperror.Internal("uploading video", err))
		return
	}

	media := &repository.NoteImage{
		UploaderID:      userID,
		ObjectKey:       objectKey,
		MimeType:        contentType,
		MediaKind:       "video",
		DurationSeconds: &duration,
		SizeBytes:       int64(len(data)),
	}
	if err := h.imageRepo.Create(r.Context(), media); err != nil {
		_ = h.storage.Delete(r.Context(), objectKey)
		response.Error(w, apperror.Internal("saving video record", err))
		return
	}

	response.Created(w, UploadResponse{
		ID:              media.ID.String(),
		MediaURL:        fmt.Sprintf("/api/media/%s", media.ID),
		MediaKind:       "video",
		MimeType:        contentType,
		DurationSeconds: &duration,
	})
}

// Serve generates a short-lived signed object URL after checking that the
// requester uploaded the media or belongs to the Memory Book containing it.
func (h *MediaHandler) Serve(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}
	imageID, err := uuid.Parse(chi.URLParam(r, "imageId"))
	if err != nil {
		response.Error(w, apperror.BadRequest("invalid image ID"))
		return
	}
	if !h.imageRepo.CanAccess(r.Context(), imageID, userID) {
		response.Error(w, apperror.Forbidden("you do not have access to this media"))
		return
	}

	img, err := h.imageRepo.FindByID(r.Context(), imageID)
	if err != nil {
		response.Error(w, apperror.NotFound("image not found"))
		return
	}

	url, err := h.storage.PresignedURL(r.Context(), img.ObjectKey, presignedExpiry)
	if err != nil {
		response.Error(w, apperror.Internal("generating image URL", err))
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}
