package friendship

import (
	"context"
	"time"

	"github.com/friend-diaries/backend/internal/domain/friendship"
	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/google/uuid"
)

type SendRequestInput struct {
	RequesterID    uuid.UUID
	AddresseeAppID string
}

type SetLabelInput struct {
	FriendshipID uuid.UUID
	UserID       uuid.UUID
	Tag          string
	Nickname     string
}

type Service struct {
	friendshipRepo   friendship.Repository
	userRepo         user.Repository
	notificationRepo notification.Repository
}

func NewService(
	friendshipRepo friendship.Repository,
	userRepo user.Repository,
	notificationRepo notification.Repository,
) *Service {
	return &Service{
		friendshipRepo:   friendshipRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *Service) SendRequest(ctx context.Context, input SendRequestInput) (*friendship.Friendship, error) {
	addressee, err := s.userRepo.FindByAppID(ctx, input.AddresseeAppID)
	if err != nil || addressee.IsDeleted() {
		return nil, apperror.NotFound("user with that app ID")
	}

	if addressee.ID == input.RequesterID {
		return nil, apperror.BadRequest("you cannot add yourself")
	}

	existing, _ := s.friendshipRepo.FindBetween(ctx, input.RequesterID, addressee.ID)
	if existing != nil {
		if existing.Status == friendship.StatusAccepted {
			return nil, apperror.Conflict("you are already friends")
		}
		return nil, apperror.Conflict("a friend request already exists")
	}

	now := time.Now().UTC()
	f := &friendship.Friendship{
		ID:          uuid.New(),
		RequesterID: input.RequesterID,
		AddresseeID: addressee.ID,
		Status:      friendship.StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.friendshipRepo.Create(ctx, f); err != nil {
		return nil, apperror.Internal("sending friend request", err)
	}

	_ = s.notificationRepo.Create(ctx, &notification.Notification{
		ID:          uuid.New(),
		RecipientID: addressee.ID,
		ActorID:     input.RequesterID,
		Type:        notification.TypeGroupInvitation, // reuse until we add a dedicated type
		ReferenceID: f.ID,
		RefType:     notification.RefGroup,
		CreatedAt:   now,
	})

	return f, nil
}

func (s *Service) Accept(ctx context.Context, friendshipID, userID uuid.UUID) error {
	f, err := s.friendshipRepo.FindByID(ctx, friendshipID)
	if err != nil {
		return apperror.NotFound("friend request")
	}

	if f.AddresseeID != userID {
		return apperror.Forbidden("only the recipient can accept")
	}

	if f.Status != friendship.StatusPending {
		return apperror.BadRequest("request is not pending")
	}

	return s.friendshipRepo.UpdateStatus(ctx, friendshipID, friendship.StatusAccepted)
}

func (s *Service) Decline(ctx context.Context, friendshipID, userID uuid.UUID) error {
	f, err := s.friendshipRepo.FindByID(ctx, friendshipID)
	if err != nil {
		return apperror.NotFound("friend request")
	}

	if f.AddresseeID != userID && f.RequesterID != userID {
		return apperror.Forbidden("access denied")
	}

	return s.friendshipRepo.Delete(ctx, friendshipID)
}

func (s *Service) Remove(ctx context.Context, friendshipID, userID uuid.UUID) error {
	f, err := s.friendshipRepo.FindByID(ctx, friendshipID)
	if err != nil {
		return apperror.NotFound("friendship")
	}

	if f.RequesterID != userID && f.AddresseeID != userID {
		return apperror.Forbidden("access denied")
	}

	return s.friendshipRepo.Delete(ctx, friendshipID)
}

func (s *Service) SetLabel(ctx context.Context, input SetLabelInput) error {
	f, err := s.friendshipRepo.FindByID(ctx, input.FriendshipID)
	if err != nil {
		return apperror.NotFound("friendship")
	}

	if f.RequesterID != input.UserID && f.AddresseeID != input.UserID {
		return apperror.Forbidden("access denied")
	}

	if f.Status != friendship.StatusAccepted {
		return apperror.BadRequest("can only label accepted friends")
	}

	return s.friendshipRepo.UpsertLabel(ctx, &friendship.Label{
		FriendshipID: input.FriendshipID,
		UserID:       input.UserID,
		Tag:          input.Tag,
		Nickname:     input.Nickname,
		UpdatedAt:    time.Now().UTC(),
	})
}

func (s *Service) GetFriends(ctx context.Context, userID uuid.UUID) ([]*friendship.FriendDetail, error) {
	friends, err := s.friendshipRepo.GetFriends(ctx, userID)
	if err != nil {
		return nil, apperror.Internal("fetching friends", err)
	}
	return friends, nil
}

func (s *Service) GetPendingRequests(ctx context.Context, userID uuid.UUID) ([]*friendship.PendingRequest, error) {
	reqs, err := s.friendshipRepo.GetPendingRequests(ctx, userID)
	if err != nil {
		return nil, apperror.Internal("fetching pending requests", err)
	}
	return reqs, nil
}
