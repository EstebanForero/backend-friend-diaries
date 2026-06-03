package dto

import (
	"time"

	"github.com/friend-diaries/backend/internal/domain/friendship"
	"github.com/google/uuid"
)

type SendFriendRequestRequest struct {
	AppID string `json:"app_id"`
}

func (r *SendFriendRequestRequest) Validate() error {
	if r.AppID == "" {
		return errRequired("app_id")
	}
	return nil
}

type SetFriendLabelRequest struct {
	Tag      string `json:"tag"`
	Nickname string `json:"nickname"`
}

type FriendResponse struct {
	FriendshipID uuid.UUID `json:"friendship_id"`
	UserID       uuid.UUID `json:"user_id"`
	DisplayName  string    `json:"display_name"`
	AppID        string    `json:"app_id"`
	Tag          string    `json:"tag"`
	Nickname     string    `json:"nickname"`
	Since        time.Time `json:"since"`
}

func ToFriendResponse(d *friendship.FriendDetail) FriendResponse {
	return FriendResponse{
		FriendshipID: d.FriendshipID,
		UserID:       d.UserID,
		DisplayName:  d.DisplayName,
		AppID:        d.AppID,
		Tag:          d.Tag,
		Nickname:     d.Nickname,
		Since:        d.Since,
	}
}

type PendingRequestResponse struct {
	FriendshipID uuid.UUID `json:"friendship_id"`
	RequesterID  uuid.UUID `json:"requester_id"`
	DisplayName  string    `json:"display_name"`
	AppID        string    `json:"app_id"`
	SentAt       time.Time `json:"sent_at"`
}

func ToPendingRequestResponse(p *friendship.PendingRequest) PendingRequestResponse {
	return PendingRequestResponse{
		FriendshipID: p.FriendshipID,
		RequesterID:  p.RequesterID,
		DisplayName:  p.DisplayName,
		AppID:        p.AppID,
		SentAt:       p.SentAt,
	}
}

type GroupMemberDetailResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AppID       string    `json:"app_id"`
	Role        string    `json:"role"`
	JoinedAt    *string   `json:"joined_at"`
}
