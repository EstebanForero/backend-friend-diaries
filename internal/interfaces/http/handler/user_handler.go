package handler

import (
	"net/http"

	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/internal/interfaces/http/middleware"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	userRepo user.Repository
}

func NewUserHandler(userRepo user.Repository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}

	u, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		response.Error(w, apperror.NotFound("user"))
		return
	}

	response.OK(w, dto.ToUserResponse(u))
}

func (h *UserHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appId")
	if appID == "" {
		response.Error(w, apperror.BadRequest("app_id is required"))
		return
	}

	u, err := h.userRepo.FindByAppID(r.Context(), appID)
	if err != nil || u.IsDeleted() {
		response.Error(w, apperror.NotFound("user"))
		return
	}

	response.OK(w, map[string]any{
		"user_id":      u.ID,
		"display_name": u.PublicDisplayName(),
		"app_id":       u.AppID,
	})
}
