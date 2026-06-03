package group

import (
	"time"

	"github.com/google/uuid"
)

const (
	FreePlanMaxMembers    = 2
	PremiumPlanMaxMembers = 10
)

type Group struct {
	ID        uuid.UUID
	Name      string
	OwnerID   uuid.UUID
	Timezone  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleMember MemberRole = "member"
)

type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationDeclined InvitationStatus = "declined"
)

type Member struct {
	GroupID          uuid.UUID
	UserID           uuid.UUID
	Role             MemberRole
	InvitationStatus InvitationStatus
	InvitedAt        time.Time
	JoinedAt         *time.Time
}

func (m *Member) IsActive() bool {
	return m.InvitationStatus == InvitationAccepted
}

// PendingInvitation is a read model for an invite the current user received.
type PendingInvitation struct {
	GroupID   uuid.UUID `json:"group_id"`
	GroupName string    `json:"group_name"`
	InvitedAt time.Time `json:"invited_at"`
}

// MemberDetail is a read model combining group_members + users data.
type MemberDetail struct {
	UserID      string     `json:"user_id"`
	DisplayName string     `json:"display_name"`
	AppID       string     `json:"app_id"`
	Role        MemberRole `json:"role"`
	JoinedAt    *string    `json:"joined_at"`
}

func (g *Group) MaxMembers(ownerIsPremium bool) int {
	if ownerIsPremium {
		return PremiumPlanMaxMembers
	}
	return FreePlanMaxMembers
}
