package reaction

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Upsert(ctx context.Context, r *Reaction) error
	Delete(ctx context.Context, noteID, userID uuid.UUID) error
	FindByNote(ctx context.Context, noteID uuid.UUID) ([]*Reaction, error)
	FindByNoteAndUser(ctx context.Context, noteID, userID uuid.UUID) (*Reaction, error)
}
