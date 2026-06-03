package friendship

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, f *Friendship) error
	FindByID(ctx context.Context, id uuid.UUID) (*Friendship, error)
	FindBetween(ctx context.Context, userA, userB uuid.UUID) (*Friendship, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	Delete(ctx context.Context, id uuid.UUID) error

	GetFriends(ctx context.Context, userID uuid.UUID) ([]*FriendDetail, error)
	GetPendingRequests(ctx context.Context, userID uuid.UUID) ([]*PendingRequest, error)

	UpsertLabel(ctx context.Context, l *Label) error
	GetLabel(ctx context.Context, friendshipID, userID uuid.UUID) (*Label, error)
}
