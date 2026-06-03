package reaction

import (
	"context"
	"time"

	"github.com/friend-diaries/backend/internal/domain/group"
	"github.com/friend-diaries/backend/internal/domain/note"
	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/friend-diaries/backend/internal/domain/reaction"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/google/uuid"
)

type UpsertInput struct {
	NoteID       uuid.UUID
	UserID       uuid.UUID
	ReactionType reaction.Type
}

type Service struct {
	reactionRepo     reaction.Repository
	noteRepo         note.Repository
	groupRepo        group.Repository
	notificationRepo notification.Repository
}

func NewService(
	reactionRepo reaction.Repository,
	noteRepo note.Repository,
	groupRepo group.Repository,
	notificationRepo notification.Repository,
) *Service {
	return &Service{
		reactionRepo:     reactionRepo,
		noteRepo:         noteRepo,
		groupRepo:        groupRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *Service) Upsert(ctx context.Context, input UpsertInput) error {
	if !reaction.IsValidType(input.ReactionType) {
		return apperror.BadRequest("invalid reaction type")
	}

	n, err := s.noteRepo.FindByID(ctx, input.NoteID)
	if err != nil {
		return apperror.NotFound("note")
	}

	if err := s.assertMembership(ctx, n.GroupID, input.UserID); err != nil {
		return err
	}

	now := time.Now().UTC()
	r := &reaction.Reaction{
		ID:           uuid.New(),
		NoteID:       input.NoteID,
		UserID:       input.UserID,
		ReactionType: input.ReactionType,
		CreatedAt:    now,
	}

	if err := s.reactionRepo.Upsert(ctx, r); err != nil {
		return apperror.Internal("upserting reaction", err)
	}

	if n.AuthorID != input.UserID {
		_ = s.notificationRepo.Create(ctx, &notification.Notification{
			ID:          uuid.New(),
			RecipientID: n.AuthorID,
			ActorID:     input.UserID,
			Type:        notification.TypeReaction,
			ReferenceID: n.ID,
			RefType:     notification.RefNote,
			Metadata:    map[string]string{"reaction_type": string(input.ReactionType)},
			CreatedAt:   now,
		})
	}

	return nil
}

func (s *Service) Remove(ctx context.Context, noteID, userID uuid.UUID) error {
	n, err := s.noteRepo.FindByID(ctx, noteID)
	if err != nil {
		return apperror.NotFound("note")
	}

	if err := s.assertMembership(ctx, n.GroupID, userID); err != nil {
		return err
	}

	return s.reactionRepo.Delete(ctx, noteID, userID)
}

func (s *Service) GetByNote(ctx context.Context, noteID, userID uuid.UUID) ([]*reaction.Reaction, error) {
	n, err := s.noteRepo.FindByID(ctx, noteID)
	if err != nil {
		return nil, apperror.NotFound("note")
	}

	if err := s.assertMembership(ctx, n.GroupID, userID); err != nil {
		return nil, err
	}

	return s.reactionRepo.FindByNote(ctx, noteID)
}

func (s *Service) assertMembership(ctx context.Context, groupID, userID uuid.UUID) error {
	m, err := s.groupRepo.FindMember(ctx, groupID, userID)
	if err != nil || !m.IsActive() {
		return apperror.Forbidden("access denied")
	}
	return nil
}
