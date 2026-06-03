package group

import (
	"context"
	"time"

	"github.com/friend-diaries/backend/internal/domain/group"
	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/google/uuid"
)

type CreateInput struct {
	Name     string
	OwnerID  uuid.UUID
	Timezone string
}

type InviteInput struct {
	GroupID   uuid.UUID
	InviterID uuid.UUID
	InviteeID uuid.UUID
}

type Service struct {
	groupRepo        group.Repository
	userRepo         user.Repository
	notificationRepo notification.Repository
}

func NewService(groupRepo group.Repository, userRepo user.Repository, notificationRepo notification.Repository) *Service {
	return &Service{groupRepo: groupRepo, userRepo: userRepo, notificationRepo: notificationRepo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*group.Group, error) {
	tz := input.Timezone
	if tz == "" {
		tz = "UTC"
	}

	now := time.Now().UTC()
	g := &group.Group{
		ID:        uuid.New(),
		Name:      input.Name,
		OwnerID:   input.OwnerID,
		Timezone:  tz,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.groupRepo.Create(ctx, g); err != nil {
		return nil, apperror.Internal("creating group", err)
	}

	joinedAt := now
	ownerMember := &group.Member{
		GroupID:          g.ID,
		UserID:           input.OwnerID,
		Role:             group.RoleOwner,
		InvitationStatus: group.InvitationAccepted,
		InvitedAt:        now,
		JoinedAt:         &joinedAt,
	}

	if err := s.groupRepo.AddMember(ctx, ownerMember); err != nil {
		return nil, apperror.Internal("adding owner as member", err)
	}

	return g, nil
}

func (s *Service) Invite(ctx context.Context, input InviteInput) error {
	g, err := s.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return apperror.NotFound("group")
	}

	inviterMember, err := s.groupRepo.FindMember(ctx, input.GroupID, input.InviterID)
	if err != nil || !inviterMember.IsActive() {
		return apperror.Forbidden("you are not a member of this group")
	}

	owner, err := s.userRepo.FindByID(ctx, g.OwnerID)
	if err != nil {
		return apperror.Internal("fetching owner", err)
	}

	count, err := s.groupRepo.CountActiveMembers(ctx, input.GroupID)
	if err != nil {
		return apperror.Internal("counting members", err)
	}

	if count >= g.MaxMembers(owner.IsPremium) {
		return apperror.Forbidden("group has reached its member limit")
	}

	invitee, err := s.userRepo.FindByID(ctx, input.InviteeID)
	if err != nil {
		return apperror.NotFound("invitee user")
	}
	if invitee.IsDeleted() {
		return apperror.NotFound("invitee user")
	}

	existing, _ := s.groupRepo.FindMember(ctx, input.GroupID, input.InviteeID)
	if existing != nil && existing.IsActive() {
		return apperror.Conflict("user is already a member")
	}

	now := time.Now().UTC()
	m := &group.Member{
		GroupID:          input.GroupID,
		UserID:           input.InviteeID,
		Role:             group.RoleMember,
		InvitationStatus: group.InvitationPending,
		InvitedAt:        now,
	}

	if err := s.groupRepo.AddMember(ctx, m); err != nil {
		return apperror.Internal("creating invitation", err)
	}

	_ = s.notificationRepo.Create(ctx, &notification.Notification{
		ID:          uuid.New(),
		RecipientID: input.InviteeID,
		ActorID:     input.InviterID,
		Type:        notification.TypeGroupInvitation,
		ReferenceID: input.GroupID,
		RefType:     notification.RefGroup,
		CreatedAt:   now,
	})

	return nil
}

func (s *Service) AcceptInvitation(ctx context.Context, groupID, userID uuid.UUID) error {
	m, err := s.groupRepo.FindMember(ctx, groupID, userID)
	if err != nil {
		return apperror.NotFound("invitation")
	}

	if m.InvitationStatus != group.InvitationPending {
		return apperror.BadRequest("no pending invitation found")
	}

	now := time.Now().UTC()
	m.InvitationStatus = group.InvitationAccepted
	m.JoinedAt = &now

	if err := s.groupRepo.UpdateMember(ctx, m); err != nil {
		return apperror.Internal("accepting invitation", err)
	}
	_ = s.notificationRepo.DeleteByRecipientReferenceType(ctx, userID, groupID, notification.TypeGroupInvitation)

	g, _ := s.groupRepo.FindByID(ctx, groupID)
	if g != nil {
		_ = s.notificationRepo.Create(ctx, &notification.Notification{
			ID:          uuid.New(),
			RecipientID: g.OwnerID,
			ActorID:     userID,
			Type:        notification.TypeInvitationAccept,
			ReferenceID: groupID,
			RefType:     notification.RefGroup,
			CreatedAt:   now,
		})
	}

	return nil
}

func (s *Service) DeclineInvitation(ctx context.Context, groupID, userID uuid.UUID) error {
	m, err := s.groupRepo.FindMember(ctx, groupID, userID)
	if err != nil {
		return apperror.NotFound("invitation")
	}

	if m.InvitationStatus != group.InvitationPending {
		return apperror.BadRequest("no pending invitation found")
	}

	m.InvitationStatus = group.InvitationDeclined
	if err := s.groupRepo.UpdateMember(ctx, m); err != nil {
		return err
	}
	_ = s.notificationRepo.DeleteByRecipientReferenceType(ctx, userID, groupID, notification.TypeGroupInvitation)
	return nil
}

func (s *Service) Leave(ctx context.Context, groupID, userID uuid.UUID) error {
	g, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return apperror.NotFound("group")
	}

	if g.OwnerID == userID {
		return apperror.BadRequest("group owner cannot leave; delete the group instead")
	}

	m, err := s.groupRepo.FindMember(ctx, groupID, userID)
	if err != nil || !m.IsActive() {
		return apperror.Forbidden("you are not a member of this group")
	}

	return s.groupRepo.RemoveMember(ctx, groupID, userID)
}

func (s *Service) GetUserGroups(ctx context.Context, userID uuid.UUID) ([]*group.Group, error) {
	groups, err := s.groupRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.Internal("fetching groups", err)
	}
	return groups, nil
}

func (s *Service) GetGroup(ctx context.Context, groupID, userID uuid.UUID) (*group.Group, error) {
	g, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return nil, apperror.NotFound("group")
	}

	member, err := s.groupRepo.FindMember(ctx, groupID, userID)
	if err != nil || !member.IsActive() {
		return nil, apperror.Forbidden("access denied")
	}

	return g, nil
}

func (s *Service) assertMembership(ctx context.Context, groupID, userID uuid.UUID) error {
	m, err := s.groupRepo.FindMember(ctx, groupID, userID)
	if err != nil || !m.IsActive() {
		return apperror.Forbidden("access denied")
	}
	return nil
}
