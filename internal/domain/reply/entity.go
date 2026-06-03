package reply

import (
	"time"

	"github.com/google/uuid"
)

type Reply struct {
	ID        uuid.UUID
	NoteID    uuid.UUID
	AuthorID  uuid.UUID
	Content   string
	IsEdited  bool
	IsDeleted bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (r *Reply) IsOwnedBy(userID uuid.UUID) bool {
	return r.AuthorID == userID
}

func (r *Reply) IsToday() bool {
	now := time.Now().UTC()
	created := r.CreatedAt.UTC()
	return now.Year() == created.Year() &&
		now.Month() == created.Month() &&
		now.Day() == created.Day()
}
