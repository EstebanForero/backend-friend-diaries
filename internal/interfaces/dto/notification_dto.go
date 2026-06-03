package dto

import (
	"time"

	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/google/uuid"
)

type NotificationResponse struct {
	ID               uuid.UUID            `json:"id"`
	RecipientID      uuid.UUID            `json:"recipient_id"`
	ActorID          uuid.UUID            `json:"actor_id"`
	ActorDisplayName string               `json:"actor_display_name"`
	Type             notification.Type    `json:"type"`
	ReferenceID      uuid.UUID            `json:"reference_id"`
	RefType          notification.RefType `json:"ref_type"`
	Metadata         map[string]string    `json:"metadata"`
	IsRead           bool                 `json:"is_read"`
	CreatedAt        time.Time            `json:"created_at"`
}

func ToNotificationResponse(n *notification.Notification) NotificationResponse {
	meta := n.Metadata
	if meta == nil {
		meta = map[string]string{}
	}
	return NotificationResponse{
		ID:               n.ID,
		RecipientID:      n.RecipientID,
		ActorID:          n.ActorID,
		ActorDisplayName: n.ActorDisplayName,
		Type:             n.Type,
		ReferenceID:      n.ReferenceID,
		RefType:          n.RefType,
		Metadata:         meta,
		IsRead:           n.IsRead,
		CreatedAt:        n.CreatedAt,
	}
}
