package notification

import (
	"context"

	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/google/uuid"
)

const defaultPageSize = 20

type Service struct {
	notificationRepo notification.Repository
}

func NewService(notificationRepo notification.Repository) *Service {
	return &Service{notificationRepo: notificationRepo}
}

func (s *Service) GetForUser(ctx context.Context, userID uuid.UUID, page int) ([]*notification.Notification, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * defaultPageSize

	ns, err := s.notificationRepo.FindByRecipient(ctx, userID, defaultPageSize, offset)
	if err != nil {
		return nil, apperror.Internal("fetching notifications", err)
	}
	return ns, nil
}

func (s *Service) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.notificationRepo.CountUnread(ctx, userID)
	if err != nil {
		return 0, apperror.Internal("counting unread notifications", err)
	}
	return count, nil
}

func (s *Service) MarkRead(ctx context.Context, notificationID, userID uuid.UUID) error {
	if err := s.notificationRepo.MarkRead(ctx, notificationID, userID); err != nil {
		return apperror.Internal("marking notification as read", err)
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	if err := s.notificationRepo.MarkAllRead(ctx, userID); err != nil {
		return apperror.Internal("marking all notifications as read", err)
	}
	return nil
}
