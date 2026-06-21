package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/friend-diaries/backend/internal/infrastructure/repository"
	"github.com/friend-diaries/backend/internal/infrastructure/storage"
	"github.com/friend-diaries/backend/internal/interfaces/http/middleware"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	maxImageSize    = 10 << 20 // 10 MB
	presignedExpiry = 24 * time.Hour
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
}

func NewMediaHandler(storage *storage.Client, imageRepo *repository.NoteImageRepository) *MediaHandler {
	return &MediaHandler{storage: storage, imageRepo: imageRepo}
}

type UploadResponse struct {
	ID       string `json:"id"`
	MediaURL string `json:"media_url"`
}

// Upload handles multipart image upload, stores in MinIO, and returns the image ID.
// The image is initially unattached (note_id = NULL) and gets linked when the note is created.
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}

	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		response.Error(w, apperror.BadRequest("image too large (max 10 MB)"))
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		response.Error(w, apperror.BadRequest("missing image field"))
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

	ext, ok := allowedMimeTypes[contentType]
	if !ok {
		response.Error(w, apperror.BadRequest(fmt.Sprintf("unsupported image type: %s", contentType)))
		return
	}

	// Resize and re-encode to WebP regardless of input format
	imageData, _, _, procErr := storage.ProcessImage(file, contentType)
	if procErr != nil {
		response.Error(w, apperror.BadRequest("could not process image: "+procErr.Error()))
		return
	}
	_ = ext // no longer needed — always .webp

	objectKey := fmt.Sprintf("images/%s/%s.webp", userID, uuid.New())
	webpSize := int64(len(imageData))

	if err := h.storage.Upload(r.Context(), objectKey, bytes.NewReader(imageData), webpSize, "image/webp"); err != nil {
		response.Error(w, apperror.Internal("uploading image", err))
		return
	}

	img := &repository.NoteImage{
		UploaderID: userID,
		ObjectKey:  objectKey,
		MimeType:   "image/webp",
		SizeBytes:  webpSize,
	}
	if err := h.imageRepo.Create(r.Context(), img); err != nil {
		// best-effort cleanup
		_ = h.storage.Delete(r.Context(), objectKey)
		response.Error(w, apperror.Internal("saving image record", err))
		return
	}

	response.Created(w, UploadResponse{
		ID:       img.ID.String(),
		MediaURL: fmt.Sprintf("/media/%s", img.ID),
	})
}

// Serve generates a presigned URL for an image and redirects to it.
// Only authenticated users can access images.
func (h *MediaHandler) Serve(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}

	imageID, err := uuid.Parse(chi.URLParam(r, "imageId"))
	if err != nil {
		response.Error(w, apperror.BadRequest("invalid image ID"))
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
