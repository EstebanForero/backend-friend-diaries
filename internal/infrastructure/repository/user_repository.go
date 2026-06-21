package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/friend-diaries/backend/internal/domain/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, app_id, is_premium, consent_given_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, u.ID, u.Email, u.PasswordHash, u.DisplayName, u.AppID, u.IsPremium, u.ConsentGivenAt, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return r.scanOne(ctx, `
		SELECT id, email, password_hash, display_name, app_id, is_premium, consent_given_at, created_at, updated_at, deleted_at
		FROM users WHERE id = $1
	`, id)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.scanOne(ctx, `
		SELECT id, email, password_hash, display_name, app_id, is_premium, consent_given_at, created_at, updated_at, deleted_at
		FROM users WHERE email = $1
	`, email)
}

func (r *UserRepository) FindByAppID(ctx context.Context, appID string) (*user.User, error) {
	return r.scanOne(ctx, `
		SELECT id, email, password_hash, display_name, app_id, is_premium, consent_given_at, created_at, updated_at, deleted_at
		FROM users WHERE app_id = $1
	`, appID)
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	u.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET email=$1, display_name=$2, is_premium=$3, updated_at=$4 WHERE id=$5
	`, u.Email, u.DisplayName, u.IsPremium, u.UpdatedAt, u.ID)
	return err
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET deleted_at=$1, updated_at=$1, display_name='Deleted User' WHERE id=$2
	`, now, id)
	return err
}

func (r *UserRepository) AppIDExists(ctx context.Context, appID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE app_id = $1)`, appID).Scan(&exists)
	return exists, err
}

func (r *UserRepository) scanOne(ctx context.Context, query string, args ...any) (*user.User, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	u := &user.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AppID,
		&u.IsPremium, &u.ConsentGivenAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("scanning user: %w", err)
	}
	return u, nil
}
