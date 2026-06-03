package note

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	AuthorID  uuid.UUID
	DiaryDate time.Time
	Content   string
	IsEdited  bool
	IsDeleted bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (n *Note) IsOwnedBy(userID uuid.UUID) bool {
	return n.AuthorID == userID
}

func (n *Note) IsToday() bool {
	now := time.Now().UTC()
	date := n.DiaryDate.UTC()
	return now.Year() == date.Year() &&
		now.Month() == date.Month() &&
		now.Day() == date.Day()
}

func (n *Note) CanEdit(userID uuid.UUID, isPremium bool) bool {
	if !n.IsOwnedBy(userID) {
		return false
	}
	if n.IsToday() {
		return true
	}
	return isPremium
}
