package handler

import (
	"encoding/json"
	"net/http"

	"github.com/friend-diaries/backend/internal/infrastructure/repository"
	"github.com/friend-diaries/backend/internal/interfaces/http/middleware"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
)

type PushHandler struct {
	repo      *repository.PushSubscriptionRepository
	vapidPubKey string
}

func NewPushHandler(repo *repository.PushSubscriptionRepository, vapidPubKey string) *PushHandler {
	return &PushHandler{repo: repo, vapidPubKey: vapidPubKey}
}

func (h *PushHandler) VAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"key": h.vapidPubKey})
}

func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}

	var body struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Endpoint == "" {
		response.Error(w, apperror.BadRequest("invalid subscription payload"))
		return
	}

	sub := &repository.PushSubscription{
		UserID:   userID,
		Endpoint: body.Endpoint,
		P256dh:   body.Keys.P256dh,
		Auth:     body.Keys.Auth,
	}
	if err := h.repo.Save(r.Context(), sub); err != nil {
		response.Error(w, apperror.Internal("saving push subscription", err))
		return
	}
	response.NoContent(w)
}

func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}

	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Endpoint == "" {
		response.Error(w, apperror.BadRequest("missing endpoint"))
		return
	}

	_ = h.repo.Delete(r.Context(), body.Endpoint, userID)
	response.NoContent(w)
}
