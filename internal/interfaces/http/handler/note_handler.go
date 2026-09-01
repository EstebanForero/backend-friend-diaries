package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	notesvc "github.com/friend-diaries/backend/internal/application/note"
	"github.com/friend-diaries/backend/internal/infrastructure/repository"
	"github.com/friend-diaries/backend/internal/infrastructure/storage"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type NoteHandler struct {
	noteService   *notesvc.Service
	imageRepo     *repository.NoteImageRepository
	storageClient *storage.Client
}

func NewNoteHandler(noteService *notesvc.Service, imageRepo *repository.NoteImageRepository, storageClient *storage.Client) *NoteHandler {
	return &NoteHandler{noteService: noteService, imageRepo: imageRepo, storageClient: storageClient}
}

func (h *NoteHandler) loadMedia(r *http.Request, nr *dto.NoteResponse) {
	if h.imageRepo == nil {
		return
	}
	imgs, err := h.imageRepo.ListByNoteID(r.Context(), nr.ID)
	if err != nil || len(imgs) == 0 {
		return
	}
	for _, img := range imgs {
		mediaURL := fmt.Sprintf("/api/media/%s", img.ID)
		// Generate a direct presigned URL when storage is available so clients
		// (especially Tauri/Android) don't need to follow a redirect.
		if h.storageClient != nil {
			if presigned, err := h.storageClient.PresignedURL(r.Context(), img.ObjectKey, 24*time.Hour); err == nil {
				mediaURL = presigned
			} else {
				slog.Warn("could not generate presigned url", "image_id", img.ID, "error", err)
			}
		}
		nr.Media = append(nr.Media, dto.NoteMediaResponse{
			ID:              img.ID,
			MediaURL:        mediaURL,
			MimeType:        img.MimeType,
			MediaKind:       img.MediaKind,
			DurationSeconds: img.DurationSeconds,
		})
	}
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	var req dto.CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}

	date := time.Now().UTC()
	if req.Date != "" {
		parsed, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			response.Error(w, apperror.BadRequest("invalid date format, use YYYY-MM-DD"))
			return
		}
		date = parsed
	}

	var clientID *uuid.UUID
	if req.ClientID != "" {
		parsed, _ := uuid.Parse(req.ClientID) // validated above
		clientID = &parsed
	}

	n, err := h.noteService.Create(r.Context(), notesvc.CreateInput{
		GroupID:  groupID,
		AuthorID: userID,
		ClientID: clientID,
		Content:  req.Content,
		Date:     date,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	// Attach any pre-uploaded images to this note
	if len(req.MediaIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(req.MediaIDs))
		for _, s := range req.MediaIDs {
			if id, err := uuid.Parse(s); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			_ = h.imageRepo.AttachToNote(r.Context(), ids, n.ID, userID)
		}
	}

	nr := dto.ToNoteResponse(n)
	h.loadMedia(r, &nr)
	response.Created(w, nr)
}

func (h *NoteHandler) CalendarSummary(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	month := r.URL.Query().Get("month")
	if month == "" {
		response.Error(w, apperror.BadRequest("month query param required (YYYY-MM)"))
		return
	}

	var authorID *uuid.UUID
	if a := r.URL.Query().Get("author"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			response.Error(w, apperror.BadRequest("invalid author id"))
			return
		}
		authorID = &id
	}

	summary, err := h.noteService.CalendarSummary(r.Context(), groupID, userID, month, authorID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, summary)
}

func (h *NoteHandler) ListByDate(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")
	dateStr := chi.URLParam(r, "date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.Error(w, apperror.BadRequest("invalid date format, use YYYY-MM-DD"))
		return
	}

	notes, err := h.noteService.GetByGroupAndDate(r.Context(), groupID, userID, date)
	if err != nil {
		response.Error(w, err)
		return
	}

	result := make([]dto.NoteResponse, 0, len(notes))
	for _, n := range notes {
		nr := dto.ToNoteResponse(n)
		h.loadMedia(r, &nr)
		result = append(result, nr)
	}

	response.OK(w, result)
}

func (h *NoteHandler) Recent(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")
	limit := 8

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			response.Error(w, apperror.BadRequest("limit must be a number"))
			return
		}
		limit = parsed
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 20 {
		limit = 20
	}

	notes, err := h.noteService.GetRecent(r.Context(), groupID, userID, limit)
	if err != nil {
		response.Error(w, err)
		return
	}

	result := make([]dto.NoteResponse, 0, len(notes))
	for _, n := range notes {
		nr := dto.ToNoteResponse(n)
		h.loadMedia(r, &nr)
		result = append(result, nr)
	}

	response.OK(w, result)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	noteID := mustUUID(r, "noteId")

	var req dto.UpdateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}

	n, err := h.noteService.Update(r.Context(), notesvc.UpdateInput{
		NoteID:  noteID,
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	nr := dto.ToNoteResponse(n)
	h.loadMedia(r, &nr)
	response.OK(w, nr)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	noteID := mustUUID(r, "noteId")

	if err := h.noteService.Delete(r.Context(), noteID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *NoteHandler) CreateShareToken(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	noteID := mustUUID(r, "noteId")

	shareToken, err := h.noteService.CreateShareToken(r.Context(), noteID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, map[string]string{"share_token": shareToken})
}

func (h *NoteHandler) GetByShareToken(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	shareToken := chi.URLParam(r, "token")

	n, err := h.noteService.GetByShareToken(r.Context(), shareToken, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	nr := dto.ToNoteResponse(n)
	h.loadMedia(r, &nr)
	response.OK(w, nr)
}

func (h *NoteHandler) Adjacent(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().UTC().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.Error(w, apperror.BadRequest("invalid date format, use YYYY-MM-DD"))
		return
	}

	var authorID *uuid.UUID
	if a := r.URL.Query().Get("author"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			authorID = &id
		}
	}

	prev, next, err := h.noteService.GetAdjacentDates(r.Context(), groupID, userID, date, authorID)
	if err != nil {
		response.Error(w, err)
		return
	}

	type Result struct {
		Prev *string `json:"prev"`
		Next *string `json:"next"`
	}
	res := Result{}
	if prev != nil {
		s := prev.Format("2006-01-02")
		res.Prev = &s
	}
	if next != nil {
		s := next.Format("2006-01-02")
		res.Next = &s
	}
	response.OK(w, res)
}

func (h *NoteHandler) Streak(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	groupID := mustUUID(r, "groupId")

	authorID := userID
	if a := r.URL.Query().Get("author"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			authorID = id
		}
	}

	current, recoverable, err := h.noteService.GetStreak(r.Context(), groupID, userID, authorID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, map[string]any{"current": current, "recoverable": recoverable})
}

var _ = uuid.Nil
