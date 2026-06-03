package handler

import (
	"encoding/json"
	"net/http"
	"time"

	notesvc "github.com/friend-diaries/backend/internal/application/note"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type NoteHandler struct {
	noteService *notesvc.Service
}

func NewNoteHandler(noteService *notesvc.Service) *NoteHandler {
	return &NoteHandler{noteService: noteService}
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

	response.Created(w, dto.ToNoteResponse(n))
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
		result = append(result, dto.ToNoteResponse(n))
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
