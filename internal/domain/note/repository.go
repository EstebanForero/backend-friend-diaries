package note

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, n *Note) error
	FindByID(ctx context.Context, id uuid.UUID) (*Note, error)
	FindByGroupAndDate(ctx context.Context, groupID uuid.UUID, date time.Time) ([]*Note, error)
	Update(ctx context.Context, n *Note) error
	SoftDelete(ctx context.Context, id uuid.UUID) error

	CreateShareToken(ctx context.Context, noteID uuid.UUID, token string, expiresAt time.Time) error
	FindByShareToken(ctx context.Context, token string) (*Note, error)

	// CalendarSummary returns a map of "YYYY-MM-DD" → count of non-deleted notes
	// for the given month. If authorID is non-nil, only that author's notes are counted.
	CalendarSummary(ctx context.Context, groupID uuid.UUID, month string, authorID *uuid.UUID) (map[string]int, error)
}
