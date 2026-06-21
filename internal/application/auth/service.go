package auth

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/friend-diaries/backend/pkg/apperror"
	"github.com/friend-diaries/backend/pkg/hash"
	"github.com/friend-diaries/backend/pkg/token"
	"github.com/google/uuid"
)

type RegisterInput struct {
	Email        string
	Password     string
	DisplayName  string
	ConsentGiven bool
}

type LoginInput struct {
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	User         *user.User
}

type Service struct {
	userRepo     user.Repository
	tokenManager *token.Manager
}

func NewService(userRepo user.Repository, tokenManager *token.Manager) *Service {
	return &Service{userRepo: userRepo, tokenManager: tokenManager}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*user.User, error) {
	existing, err := s.userRepo.FindByEmail(ctx, strings.ToLower(input.Email))
	if err == nil && existing != nil {
		return nil, apperror.Conflict("email already registered")
	}

	passwordHash, err := hash.Password(input.Password)
	if err != nil {
		return nil, apperror.Internal("hashing password", err)
	}

	appID, err := s.generateUniqueAppID(ctx, input.DisplayName)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	u := &user.User{
		ID:             uuid.New(),
		Email:          strings.ToLower(input.Email),
		PasswordHash:   passwordHash,
		DisplayName:    input.DisplayName,
		AppID:          appID,
		IsPremium:      true, // all new users start as premium
		ConsentGivenAt: &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, apperror.Internal("creating user", err)
	}

	return u, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*TokenPair, error) {
	u, err := s.userRepo.FindByEmail(ctx, strings.ToLower(input.Email))
	if err != nil {
		return nil, apperror.Unauthorized("invalid credentials")
	}

	if u.IsDeleted() {
		return nil, apperror.Unauthorized("account has been deleted")
	}

	if !hash.VerifyPassword(u.PasswordHash, input.Password) {
		return nil, apperror.Unauthorized("invalid credentials")
	}

	return s.issueTokenPair(u)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.tokenManager.Validate(refreshToken, token.RefreshToken)
	if err != nil {
		return nil, apperror.Unauthorized("invalid refresh token")
	}

	u, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, apperror.Unauthorized("user not found")
	}

	if u.IsDeleted() {
		return nil, apperror.Unauthorized("account has been deleted")
	}

	return s.issueTokenPair(u)
}

func (s *Service) issueTokenPair(u *user.User) (*TokenPair, error) {
	accessToken, err := s.tokenManager.GenerateAccess(u.ID)
	if err != nil {
		return nil, apperror.Internal("generating access token", err)
	}

	refreshToken, err := s.tokenManager.GenerateRefresh(u.ID)
	if err != nil {
		return nil, apperror.Internal("generating refresh token", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         u,
	}, nil
}

func (s *Service) generateUniqueAppID(ctx context.Context, displayName string) (string, error) {
	base := sanitizeDisplayName(displayName)
	if base == "" {
		base = "user"
	}

	for range 10 {
		candidate := fmt.Sprintf("%s#%04d", base, rand.IntN(9000)+1000)
		exists, err := s.userRepo.AppIDExists(ctx, candidate)
		if err != nil {
			return "", apperror.Internal("checking app ID uniqueness", err)
		}
		if !exists {
			return candidate, nil
		}
	}

	return fmt.Sprintf("%s#%s", base, uuid.NewString()[:8]), nil
}

func sanitizeDisplayName(name string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}
	s := result.String()
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
