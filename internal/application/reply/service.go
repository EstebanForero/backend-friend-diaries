package reply

import (
	"context"
	"time"

	"github.com/friend-diaries/backend/internal/domain/group"
	"github.com/friend-diaries/backend/internal/domain/note"
	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/friend-diaries/backend/internal/domain/reply"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/google/uuid"
)

type CreateInput struct {
	NoteID   uuid.UUID
	AuthorID uuid.UUID
	Content  string
}

type UpdateInput struct {
	ReplyID uuid.UUID
	UserID  uuid.UUID
	Content string
}

type Service struct {
	replyRepo        reply.Repository
	noteRepo         note.Repository
	groupRepo        group.Repository
	notificationRepo notification.Repository
}

func NewService(
	replyRepo reply.Repository,
	noteRepo note.Repository,
	groupRepo group.Repository,
	notificationRepo notification.Repository,
) *Service {
	return &Service{
		replyRepo:        replyRepo,
		noteRepo:         noteRepo,
		groupRepo:        groupRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*reply.Reply, error) {
	n, err := s.noteRepo.FindByID(ctx, input.NoteID)
	if err != nil {
		return nil, apperror.NotFound("note")
	}

	if err := s.assertMembership(ctx, n.GroupID, input.AuthorID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	r := &reply.Reply{
		ID:        uuid.New(),
		NoteID:    input.NoteID,
		AuthorID:  input.AuthorID,
		Content:   input.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.replyRepo.Create(ctx, r); err != nil {
		return nil, apperror.Internal("creating reply", err)
	}

	if n.AuthorID != input.AuthorID {
		_ = s.notificationRepo.Create(ctx, &notification.Notification{
			ID:          uuid.New(),
			RecipientID: n.AuthorID,
			ActorID:     input.AuthorID,
			Type:        notification.TypeReply,
			ReferenceID: r.ID,
			RefType:     notification.RefReply,
			CreatedAt:   now,
		})
	}

	return r, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (*reply.Reply, error) {
	r, err := s.replyRepo.FindByID(ctx, input.ReplyID)
	if err != nil {
		return nil, apperror.NotFound("reply")
	}

	n, err := s.noteRepo.FindByID(ctx, r.NoteID)
	if err != nil {
		return nil, apperror.Internal("fetching parent note", err)
	}

	if err := s.assertMembership(ctx, n.GroupID, input.UserID); err != nil {
		return nil, err
	}

	if !r.IsOwnedBy(input.UserID) {
		return nil, apperror.Forbidden("you can only edit your own replies")
	}

	if !r.IsToday() {
		return nil, apperror.Forbidden("you can only edit today's replies")
	}

	r.Content = input.Content
	r.IsEdited = true
	r.UpdatedAt = time.Now().UTC()

	if err := s.replyRepo.Update(ctx, r); err != nil {
		return nil, apperror.Internal("updating reply", err)
	}

	return r, nil
}

func (s *Service) Delete(ctx context.Context, replyID, userID uuid.UUID) error {
	r, err := s.replyRepo.FindByID(ctx, replyID)
	if err != nil {
		return apperror.NotFound("reply")
	}

	n, err := s.noteRepo.FindByID(ctx, r.NoteID)
	if err != nil {
		return apperror.Internal("fetching parent note", err)
	}

	if err := s.assertMembership(ctx, n.GroupID, userID); err != nil {
		return err
	}

	if !r.IsOwnedBy(userID) {
		return apperror.Forbidden("you can only delete your own replies")
	}

	return s.replyRepo.SoftDelete(ctx, replyID)
}

func (s *Service) GetByNote(ctx context.Context, noteID, userID uuid.UUID) ([]*reply.Reply, error) {
	n, err := s.noteRepo.FindByID(ctx, noteID)
	if err != nil {
		return nil, apperror.NotFound("note")
	}

	if err := s.assertMembership(ctx, n.GroupID, userID); err != nil {
		return nil, err
	}

	return s.replyRepo.FindByNote(ctx, noteID)
}

func (s *Service) assertMembership(ctx context.Context, groupID, userID uuid.UUID) error {
	m, err := s.groupRepo.FindMember(ctx, groupID, userID)
	if err != nil || !m.IsActive() {
		return apperror.Forbidden("access denied")
	}
	return nil
}
