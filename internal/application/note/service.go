package note

import (
	"context"
	"time"

	"github.com/friend-diaries/backend/internal/domain/group"
	"github.com/friend-diaries/backend/internal/domain/note"
	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/google/uuid"
)

type CreateInput struct {
	GroupID  uuid.UUID
	AuthorID uuid.UUID
	Content  string
	Date     time.Time
}

type UpdateInput struct {
	NoteID  uuid.UUID
	UserID  uuid.UUID
	Content string
}

type Service struct {
	noteRepo         note.Repository
	groupRepo        group.Repository
	userRepo         user.Repository
	notificationRepo notification.Repository
}

func NewService(
	noteRepo note.Repository,
	groupRepo group.Repository,
	userRepo user.Repository,
	notificationRepo notification.Repository,
) *Service {
	return &Service{
		noteRepo:         noteRepo,
		groupRepo:        groupRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*note.Note, error) {
	if err := s.assertMembership(ctx, input.GroupID, input.AuthorID); err != nil {
		return nil, err
	}

	if input.Date.After(time.Now().UTC()) {
		return nil, apperror.BadRequest("cannot create notes for future dates")
	}

	now := time.Now().UTC()
	n := &note.Note{
		ID:        uuid.New(),
		GroupID:   input.GroupID,
		AuthorID:  input.AuthorID,
		DiaryDate: input.Date.Truncate(24 * time.Hour),
		Content:   input.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.noteRepo.Create(ctx, n); err != nil {
		return nil, apperror.Internal("creating note", err)
	}

	s.notifyMembers(ctx, n)

	return n, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (*note.Note, error) {
	n, err := s.noteRepo.FindByID(ctx, input.NoteID)
	if err != nil {
		return nil, apperror.NotFound("note")
	}

	if err := s.assertMembership(ctx, n.GroupID, input.UserID); err != nil {
		return nil, err
	}

	u, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.Internal("fetching user", err)
	}

	if !n.CanEdit(input.UserID, u.IsPremium) {
		return nil, apperror.Forbidden("you cannot edit this note")
	}

	n.Content = input.Content
	n.IsEdited = true
	n.UpdatedAt = time.Now().UTC()

	if err := s.noteRepo.Update(ctx, n); err != nil {
		return nil, apperror.Internal("updating note", err)
	}

	return n, nil
}

func (s *Service) Delete(ctx context.Context, noteID, userID uuid.UUID) error {
	n, err := s.noteRepo.FindByID(ctx, noteID)
	if err != nil {
		return apperror.NotFound("note")
	}

	if err := s.assertMembership(ctx, n.GroupID, userID); err != nil {
		return err
	}

	if !n.IsOwnedBy(userID) {
		return apperror.Forbidden("you can only delete your own notes")
	}

	if !n.IsToday() {
		u, err := s.userRepo.FindByID(ctx, userID)
		if err != nil || !u.IsPremium {
			return apperror.Forbidden("free users cannot delete previous notes")
		}
	}

	return s.noteRepo.SoftDelete(ctx, noteID)
}

func (s *Service) GetByGroupAndDate(ctx context.Context, groupID, userID uuid.UUID, date time.Time) ([]*note.Note, error) {
	if err := s.assertMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}

	notes, err := s.noteRepo.FindByGroupAndDate(ctx, groupID, date)
	if err != nil {
		return nil, apperror.Internal("fetching notes", err)
	}

	return notes, nil
}

func (s *Service) CreateShareToken(ctx context.Context, noteID, userID uuid.UUID) (string, error) {
	n, err := s.noteRepo.FindByID(ctx, noteID)
	if err != nil {
		return "", apperror.NotFound("note")
	}

	if err := s.assertMembership(ctx, n.GroupID, userID); err != nil {
		return "", err
	}

	shareToken := uuid.NewString()
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)

	if err := s.noteRepo.CreateShareToken(ctx, noteID, shareToken, expiresAt); err != nil {
		return "", apperror.Internal("creating share token", err)
	}

	return shareToken, nil
}

func (s *Service) GetByShareToken(ctx context.Context, token string, userID uuid.UUID) (*note.Note, error) {
	n, err := s.noteRepo.FindByShareToken(ctx, token)
	if err != nil {
		return nil, apperror.NotFound("shared note")
	}

	if err := s.assertMembership(ctx, n.GroupID, userID); err != nil {
		return nil, apperror.Forbidden("you do not have permission to view this note")
	}

	return n, nil
}

func (s *Service) assertMembership(ctx context.Context, groupID, userID uuid.UUID) error {
	m, err := s.groupRepo.FindMember(ctx, groupID, userID)
	if err != nil || !m.IsActive() {
		return apperror.Forbidden("access denied")
	}
	return nil
}

func (s *Service) notifyMembers(ctx context.Context, n *note.Note) {
	members, err := s.groupRepo.FindActiveMembers(ctx, n.GroupID)
	if err != nil {
		return
	}

	now := time.Now().UTC()
	for _, m := range members {
		if m.UserID == n.AuthorID {
			continue
		}
		_ = s.notificationRepo.Create(ctx, &notification.Notification{
			ID:          uuid.New(),
			RecipientID: m.UserID,
			ActorID:     n.AuthorID,
			Type:        notification.TypeNewNote,
			ReferenceID: n.ID,
			RefType:     notification.RefNote,
			CreatedAt:   now,
		})
	}
}
