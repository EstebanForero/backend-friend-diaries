package dto

import (
	"time"

	"github.com/friend-diaries/backend/internal/domain/group"
	"github.com/google/uuid"
)

type CreateGroupRequest struct {
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}

func (r *CreateGroupRequest) Validate() error {
	if r.Name == "" {
		return errRequired("name")
	}
	if len(r.Name) > 100 {
		return errMaxLength("name", 100)
	}
	return nil
}

type InviteRequest struct {
	InviteeID uuid.UUID `json:"invitee_id"`
}

type GroupResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Timezone  string    `json:"timezone"`
	CreatedAt time.Time `json:"created_at"`
}

func ToGroupResponse(g *group.Group) GroupResponse {
	return GroupResponse{
		ID:        g.ID,
		Name:      g.Name,
		OwnerID:   g.OwnerID,
		Timezone:  g.Timezone,
		CreatedAt: g.CreatedAt,
	}
}

type MemberResponse struct {
	GroupID          uuid.UUID          `json:"group_id"`
	UserID           uuid.UUID          `json:"user_id"`
	Role             group.MemberRole   `json:"role"`
	InvitationStatus group.InvitationStatus `json:"invitation_status"`
	JoinedAt         *time.Time         `json:"joined_at"`
}

func ToMemberResponse(m *group.Member) MemberResponse {
	return MemberResponse{
		GroupID:          m.GroupID,
		UserID:           m.UserID,
		Role:             m.Role,
		InvitationStatus: m.InvitationStatus,
		JoinedAt:         m.JoinedAt,
	}
}
