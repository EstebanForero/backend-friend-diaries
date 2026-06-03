package handler

import (
	"encoding/json"
	"net/http"

	friendsvc "github.com/friend-diaries/backend/internal/application/friendship"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type FriendshipHandler struct {
	svc *friendsvc.Service
}

func NewFriendshipHandler(svc *friendsvc.Service) *FriendshipHandler {
	return &FriendshipHandler{svc: svc}
}

func (h *FriendshipHandler) SendRequest(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	var req dto.SendFriendRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, err)
		return
	}

	f, err := h.svc.SendRequest(r.Context(), friendsvc.SendRequestInput{
		RequesterID:    userID,
		AddresseeAppID: req.AppID,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, map[string]string{"friendship_id": f.ID.String()})
}

func (h *FriendshipHandler) Accept(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	fid := parseFriendshipID(r)

	if err := h.svc.Accept(r.Context(), fid, userID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *FriendshipHandler) Decline(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	fid := parseFriendshipID(r)

	if err := h.svc.Decline(r.Context(), fid, userID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *FriendshipHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	fid := parseFriendshipID(r)

	if err := h.svc.Remove(r.Context(), fid, userID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *FriendshipHandler) SetLabel(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	fid := parseFriendshipID(r)

	var req dto.SetFriendLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	if err := h.svc.SetLabel(r.Context(), friendsvc.SetLabelInput{
		FriendshipID: fid,
		UserID:       userID,
		Tag:          req.Tag,
		Nickname:     req.Nickname,
	}); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *FriendshipHandler) ListFriends(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	friends, err := h.svc.GetFriends(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	result := make([]dto.FriendResponse, 0, len(friends))
	for _, f := range friends {
		result = append(result, dto.ToFriendResponse(f))
	}
	response.OK(w, result)
}

func (h *FriendshipHandler) ListRequests(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	reqs, err := h.svc.GetPendingRequests(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	result := make([]dto.PendingRequestResponse, 0, len(reqs))
	for _, p := range reqs {
		result = append(result, dto.ToPendingRequestResponse(p))
	}
	response.OK(w, result)
}

func parseFriendshipID(r *http.Request) uuid.UUID {
	id, _ := uuid.Parse(chi.URLParam(r, "friendshipId"))
	return id
}
