package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	notesvc "github.com/friend-diaries/backend/internal/application/note"
	"github.com/friend-diaries/backend/internal/infrastructure/repository"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type NoteHandler struct {
	noteService *notesvc.Service
	imageRepo   *repository.NoteImageRepository
}

func NewNoteHandler(noteService *notesvc.Service, imageRepo *repository.NoteImageRepository) *NoteHandler {
	return &NoteHandler{noteService: noteService, imageRepo: imageRepo}
}

func (h *NoteHandler) enrichWithImages(ctx interface{ Value(any) any }, notes []dto.NoteResponse) []dto.NoteResponse {
	// noop ctx trick — we receive the real context in callers
	return notes
}

func (h *NoteHandler) loadImages(r *http.Request, nr *dto.NoteResponse) {
	if h.imageRepo == nil {
		return
	}
	imgs, err := h.imageRepo.ListByNoteID(r.Context(), nr.ID)
	if err != nil || len(imgs) == 0 {
		return
	}
	for _, img := range imgs {
		nr.Images = append(nr.Images, dto.NoteImageResponse{
			ID:       img.ID,
			MediaURL: fmt.Sprintf("/api/media/%s", img.ID),
			MimeType: img.MimeType,
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

	n, err := h.noteService.Create(r.Context(), notesvc.CreateInput{
		GroupID:  groupID,
		AuthorID: userID,
		Content:  req.Content,
		Date:     date,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	// Attach any pre-uploaded images to this note
	if len(req.ImageIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(req.ImageIDs))
		for _, s := range req.ImageIDs {
			if id, err := uuid.Parse(s); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			_ = h.imageRepo.AttachToNote(r.Context(), ids, n.ID, userID)
		}
	}

	nr := dto.ToNoteResponse(n)
	h.loadImages(r, &nr)
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
		h.loadImages(r, &nr)
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

	response.OK(w, dto.ToNoteResponse(n))
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

	response.OK(w, dto.ToNoteResponse(n))
}

var _ = uuid.Nil
