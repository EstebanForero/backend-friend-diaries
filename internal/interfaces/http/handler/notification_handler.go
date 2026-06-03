package handler

import (
	"net/http"
	"strconv"

	notisvc "github.com/friend-diaries/backend/internal/application/notification"
	"github.com/friend-diaries/backend/internal/interfaces/dto"
	"github.com/friend-diaries/backend/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notificationService *notisvc.Service
}

func NewNotificationHandler(notificationService *notisvc.Service) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	notifications, err := h.notificationService.GetForUser(r.Context(), userID, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	unread, _ := h.notificationService.CountUnread(r.Context(), userID)

	result := make([]dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		result = append(result, dto.ToNotificationResponse(n))
	}

	response.OK(w, map[string]any{
		"notifications": result,
		"unread_count":  unread,
	})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	notifID, err := uuid.Parse(chi.URLParam(r, "notificationId"))
	if err != nil {
		response.Error(w, err)
		return
	}

	if err := h.notificationService.MarkRead(r.Context(), notifID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	if err := h.notificationService.MarkAllRead(r.Context(), userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}
