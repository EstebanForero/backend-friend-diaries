package friendship

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
)

type Friendship struct {
	ID          uuid.UUID
	RequesterID uuid.UUID
	AddresseeID uuid.UUID
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// OtherUserID returns the ID of the other participant from the given user's perspective.
func (f *Friendship) OtherUserID(userID uuid.UUID) uuid.UUID {
	if f.RequesterID == userID {
		return f.AddresseeID
	}
	return f.RequesterID
}

type Label struct {
	FriendshipID uuid.UUID
	UserID       uuid.UUID
	Tag          string
	Nickname     string
	UpdatedAt    time.Time
}

// FriendDetail is a read model combining friendship + the other user's info + label.
type FriendDetail struct {
	FriendshipID uuid.UUID
	UserID       uuid.UUID
	DisplayName  string
	AppID        string
	Tag          string
	Nickname     string
	Since        time.Time
}

// PendingRequest is a read model for an incoming friend request.
type PendingRequest struct {
	FriendshipID uuid.UUID
	RequesterID  uuid.UUID
	DisplayName  string
	AppID        string
	SentAt       time.Time
}
