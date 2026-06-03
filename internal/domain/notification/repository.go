package notification

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, n *Notification) error
	FindByRecipient(ctx context.Context, recipientID uuid.UUID, limit, offset int) ([]*Notification, error)
	CountUnread(ctx context.Context, recipientID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, id, recipientID uuid.UUID) error
	MarkAllRead(ctx context.Context, recipientID uuid.UUID) error
}
