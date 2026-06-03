package handler

import (
	"encoding/json"
	"net/http"

	reactsvc "github.com/friend-diaries/backend/internal/application/reaction"
	"github.com/friend-diaries/backend/internal/domain/reaction"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
)

type ReactionHandler struct {
	reactionService *reactsvc.Service
}

func NewReactionHandler(reactionService *reactsvc.Service) *ReactionHandler {
	return &ReactionHandler{reactionService: reactionService}
}

func (h *ReactionHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	noteID := mustUUID(r, "noteId")

	var req dto.UpsertReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := h.reactionService.Upsert(r.Context(), reactsvc.UpsertInput{
		NoteID:       noteID,
		UserID:       userID,
		ReactionType: reaction.Type(req.ReactionType),
	}); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *ReactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	noteID := mustUUID(r, "noteId")

	if err := h.reactionService.Remove(r.Context(), noteID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *ReactionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	noteID := mustUUID(r, "noteId")

	reactions, err := h.reactionService.GetByNote(r.Context(), noteID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	result := make([]dto.ReactionResponse, 0, len(reactions))
	for _, react := range reactions {
		result = append(result, dto.ToReactionResponse(react))
	}

	response.OK(w, result)
}
