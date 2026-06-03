package notification

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeNewNote          Type = "new_note"
	TypeReaction         Type = "reaction"
	TypeReply            Type = "reply"
	TypeGroupInvitation  Type = "group_invitation"
	TypeInvitationAccept Type = "invitation_accepted"
)

type RefType string

const (
	RefNote  RefType = "note"
	RefGroup RefType = "group"
	RefReply RefType = "reply"
)

type Notification struct {
	ID          uuid.UUID
	RecipientID uuid.UUID
	ActorID     uuid.UUID
	Type        Type
	ReferenceID uuid.UUID
	RefType     RefType
	Metadata    map[string]string
	IsRead      bool
	CreatedAt   time.Time
}
