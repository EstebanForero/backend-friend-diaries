package handler

import (
	"encoding/json"
	"net/http"

	replysvc "github.com/friend-diaries/backend/internal/application/reply"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
)

type ReplyHandler struct {
	replyService *replysvc.Service
}

func NewReplyHandler(replyService *replysvc.Service) *ReplyHandler {
	return &ReplyHandler{replyService: replyService}
}

func (h *ReplyHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	noteID := mustUUID(r, "noteId")

	var req dto.CreateReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}

	rep, err := h.replyService.Create(r.Context(), replysvc.CreateInput{
		NoteID:   noteID,
		AuthorID: userID,
		Content:  req.Content,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, dto.ToReplyResponse(rep))
}

func (h *ReplyHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	noteID := mustUUID(r, "noteId")

	replies, err := h.replyService.GetByNote(r.Context(), noteID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	result := make([]dto.ReplyResponse, 0, len(replies))
	for _, rep := range replies {
		result = append(result, dto.ToReplyResponse(rep))
	}

	response.OK(w, result)
}

func (h *ReplyHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	replyID := mustUUID(r, "replyId")

	var req dto.UpdateReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}

	rep, err := h.replyService.Update(r.Context(), replysvc.UpdateInput{
		ReplyID: replyID,
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, dto.ToReplyResponse(rep))
}

func (h *ReplyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	replyID := mustUUID(r, "replyId")

	if err := h.replyService.Delete(r.Context(), replyID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}
