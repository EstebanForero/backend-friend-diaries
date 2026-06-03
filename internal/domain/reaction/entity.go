package reaction

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Type string

const (
	TypeLove    Type = "love"
	TypeFunny   Type = "funny"
	TypeSad     Type = "sad"
	TypeSupport Type = "support"
	TypeSeen    Type = "seen"
	TypeFav     Type = "favorite"
)

func IsValidType(t Type) bool {
	value := strings.TrimSpace(string(t))
	return value != "" && utf8.RuneCountInString(value) <= 16
}

type Reaction struct {
	ID           uuid.UUID
	NoteID       uuid.UUID
	UserID       uuid.UUID
	ReactionType Type
	CreatedAt    time.Time
}
