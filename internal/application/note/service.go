package note

import (
	"context"
	"time"

	"github.com/friend-diaries/backend/internal/domain/group"
	"github.com/friend-diaries/backend/internal/domain/note"
	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/friend-diaries/backend/internal/infrastructure/push"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/google/uuid"
)

type CreateInput struct {
	GroupID  uuid.UUID
	AuthorID uuid.UUID
	ClientID *uuid.UUID
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
	pushSvc          *push.Service
}

func NewService(
	noteRepo note.Repository,
	groupRepo group.Repository,
	userRepo user.Repository,
	notificationRepo notification.Repository,
	pushSvc *push.Service,
) *Service {
	return &Service{
		noteRepo:         noteRepo,
		groupRepo:        groupRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		pushSvc:          pushSvc,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*note.Note, error) {
	if err := s.assertMembership(ctx, input.GroupID, input.AuthorID); err != nil {
		return nil, err
	}

	// Offline clients retry until they receive a response. Return the note from
	// the first successful attempt so a lost response can never create a duplicate.
	if input.ClientID != nil {
		if existing, err := s.noteRepo.FindByClientID(ctx, input.AuthorID, *input.ClientID); err == nil {
			return existing, nil
		}
	}

	// Validate the date against "today" in the diary's timezone, not UTC.
	// A diary in UTC+9 may have a date one day ahead of UTC, which would
	// be wrongly rejected if we compared against time.Now().UTC().
	g, err := s.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, apperror.Internal("fetching group", err)
	}
	loc, locErr := time.LoadLocation(g.Timezone)
	if locErr != nil {
		loc = time.UTC
	}
	nowInTZ := time.Now().In(loc)
	todayInTZ := time.Date(nowInTZ.Year(), nowInTZ.Month(), nowInTZ.Day(), 0, 0, 0, 0, loc)
	// input.Date is parsed from a "YYYY-MM-DD" string; treat its Y/M/D as the diary date.
	inputDay := time.Date(input.Date.Year(), input.Date.Month(), input.Date.Day(), 0, 0, 0, 0, loc)
	if inputDay.After(todayInTZ) {
		return nil, apperror.BadRequest("cannot create notes for future dates")
	}

	now := time.Now().UTC()
	n := &note.Note{
		ID:        uuid.New(),
		ClientID:  input.ClientID,
		GroupID:   input.GroupID,
		AuthorID:  input.AuthorID,
		DiaryDate: time.Date(input.Date.Year(), input.Date.Month(), input.Date.Day(), 0, 0, 0, 0, time.UTC),
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

func (s *Service) GetRecent(ctx context.Context, groupID, userID uuid.UUID, limit int) ([]*note.Note, error) {
	if err := s.assertMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}

	notes, err := s.noteRepo.FindRecentByGroup(ctx, groupID, limit)
	if err != nil {
		return nil, apperror.Internal("fetching recent notes", err)
	}

	return notes, nil
}

func (s *Service) CalendarSummary(ctx context.Context, groupID, userID uuid.UUID, month string, authorID *uuid.UUID) (map[string]int, error) {
	if err := s.assertMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}
	return s.noteRepo.CalendarSummary(ctx, groupID, month, authorID)
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

func (s *Service) GetAdjacentDates(ctx context.Context, groupID, userID uuid.UUID, currentDate time.Time, authorID *uuid.UUID) (prev, next *time.Time, err error) {
	if err := s.assertMembership(ctx, groupID, userID); err != nil {
		return nil, nil, err
	}
	return s.noteRepo.GetAdjacentDates(ctx, groupID, currentDate, authorID)
}

func (s *Service) GetStreak(ctx context.Context, groupID, userID, authorID uuid.UUID) (current int, recoverable bool, err error) {
	if err := s.assertMembership(ctx, groupID, userID); err != nil {
		return 0, false, err
	}
	return s.noteRepo.GetStreak(ctx, groupID, authorID)
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

	author, _ := s.userRepo.FindByID(ctx, n.AuthorID)
	authorName := "Someone"
	if author != nil {
		authorName = author.DisplayName
	}

	now := time.Now().UTC()
	var recipientIDs []uuid.UUID
	for _, m := range members {
		if m.UserID == n.AuthorID {
			continue
		}
		recipientIDs = append(recipientIDs, m.UserID)
		_ = s.notificationRepo.Create(ctx, &notification.Notification{
			ID:          uuid.New(),
			RecipientID: m.UserID,
			ActorID:     n.AuthorID,
			Type:        notification.TypeNewNote,
			ReferenceID: n.ID,
			RefType:     notification.RefNote,
			Metadata: map[string]string{
				"group_id": n.GroupID.String(),
				"note_id":  n.ID.String(),
				"date":     n.DiaryDate.Format("2006-01-02"),
			},
			CreatedAt: now,
		})
	}

	if s.pushSvc != nil && len(recipientIDs) > 0 {
		body := n.Content
		if len(body) > 120 {
			body = body[:120] + "…"
		}
		s.pushSvc.SendToUsers(ctx, recipientIDs, push.Payload{
			Title: authorName + " wrote in the diary",
			Body:  body,
			URL:   "/app/groups/" + n.GroupID.String() + "?date=" + n.DiaryDate.Format("2006-01-02"),
		})
	}
}
