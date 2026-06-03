package reply

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, r *Reply) error
	FindByID(ctx context.Context, id uuid.UUID) (*Reply, error)
	FindByNote(ctx context.Context, noteID uuid.UUID) ([]*Reply, error)
	Update(ctx context.Context, r *Reply) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
