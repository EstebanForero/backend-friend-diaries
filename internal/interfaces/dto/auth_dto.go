package dto

import (
	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (r *RegisterRequest) Validate() error {
	if r.Email == "" {
		return errRequired("email")
	}
	if len(r.Password) < 8 {
		return errMinLength("password", 8)
	}
	if r.DisplayName == "" {
		return errRequired("display_name")
	}
	if len(r.DisplayName) > 50 {
		return errMaxLength("display_name", 50)
	}
	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return errRequired("email")
	}
	if r.Password == "" {
		return errRequired("password")
	}
	return nil
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AppID       string    `json:"app_id"`
	IsPremium   bool      `json:"is_premium"`
}

func ToUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AppID:       u.AppID,
		IsPremium:   u.IsPremium,
	}
}

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	User        UserResponse `json:"user"`
}
