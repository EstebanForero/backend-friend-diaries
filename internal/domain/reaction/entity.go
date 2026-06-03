package reaction

import (
	"time"

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

var AllTypes = []Type{TypeLove, TypeFunny, TypeSad, TypeSupport, TypeSeen, TypeFav}

func IsValidType(t Type) bool {
	for _, v := range AllTypes {
		if v == t {
			return true
		}
	}
	return false
}

type Reaction struct {
	ID           uuid.UUID
	NoteID       uuid.UUID
	UserID       uuid.UUID
	ReactionType Type
	CreatedAt    time.Time
}
