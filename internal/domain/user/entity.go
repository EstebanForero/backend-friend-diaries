package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID
	Email          string
	PasswordHash   string
	DisplayName    string
	AppID          string
	IsPremium      bool
	ConsentGivenAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

func (u *User) PublicDisplayName() string {
	if u.IsDeleted() {
		return "Deleted User"
	}
	return u.DisplayName
}
