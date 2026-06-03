package group

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, g *Group) error
	FindByID(ctx context.Context, id uuid.UUID) (*Group, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*Group, error)
	Update(ctx context.Context, g *Group) error
	Delete(ctx context.Context, id uuid.UUID) error

	AddMember(ctx context.Context, m *Member) error
	FindMember(ctx context.Context, groupID, userID uuid.UUID) (*Member, error)
	UpdateMember(ctx context.Context, m *Member) error
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error
	CountActiveMembers(ctx context.Context, groupID uuid.UUID) (int, error)
	FindActiveMembers(ctx context.Context, groupID uuid.UUID) ([]*Member, error)
	FindActiveMembersWithDetails(ctx context.Context, groupID uuid.UUID) ([]*MemberDetail, error)
	FindPendingInvitationsByUser(ctx context.Context, userID uuid.UUID) ([]*Member, error)
	FindPendingInvitationsWithDetails(ctx context.Context, userID uuid.UUID) ([]*PendingInvitation, error)
}
