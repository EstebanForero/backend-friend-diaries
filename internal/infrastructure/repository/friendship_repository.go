package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/friend-diaries/backend/internal/domain/friendship"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FriendshipRepository struct {
	pool *pgxpool.Pool
}

func NewFriendshipRepository(pool *pgxpool.Pool) *FriendshipRepository {
	return &FriendshipRepository{pool: pool}
}

func (r *FriendshipRepository) Create(ctx context.Context, f *friendship.Friendship) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO friendships (id, requester_id, addressee_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, f.ID, f.RequesterID, f.AddresseeID, f.Status, f.CreatedAt, f.UpdatedAt)
	return err
}

func (r *FriendshipRepository) FindByID(ctx context.Context, id uuid.UUID) (*friendship.Friendship, error) {
	f := &friendship.Friendship{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, requester_id, addressee_id, status, created_at, updated_at
		FROM friendships WHERE id = $1
	`, id).Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("friendship not found")
		}
		return nil, err
	}
	return f, nil
}

func (r *FriendshipRepository) FindBetween(ctx context.Context, userA, userB uuid.UUID) (*friendship.Friendship, error) {
	f := &friendship.Friendship{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, requester_id, addressee_id, status, created_at, updated_at
		FROM friendships
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
	`, userA, userB).Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("friendship not found")
		}
		return nil, err
	}
	return f, nil
}

func (r *FriendshipRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status friendship.Status) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE friendships SET status = $1, updated_at = $2 WHERE id = $3
	`, status, time.Now().UTC(), id)
	return err
}

func (r *FriendshipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM friendships WHERE id = $1`, id)
	return err
}

func (r *FriendshipRepository) GetFriends(ctx context.Context, userID uuid.UUID) ([]*friendship.FriendDetail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			f.id,
			CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END AS friend_id,
			u.display_name,
			u.app_id,
			COALESCE(fl.tag, '')      AS tag,
			COALESCE(fl.nickname, '') AS nickname,
			f.created_at
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END
		LEFT JOIN friend_labels fl ON fl.friendship_id = f.id AND fl.user_id = $1
		WHERE (f.requester_id = $1 OR f.addressee_id = $1)
		  AND f.status = 'accepted'
		  AND u.deleted_at IS NULL
		ORDER BY COALESCE(NULLIF(fl.nickname, ''), u.display_name)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*friendship.FriendDetail
	for rows.Next() {
		d := &friendship.FriendDetail{}
		if err := rows.Scan(&d.FriendshipID, &d.UserID, &d.DisplayName, &d.AppID,
			&d.Tag, &d.Nickname, &d.Since); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *FriendshipRepository) GetPendingRequests(ctx context.Context, userID uuid.UUID) ([]*friendship.PendingRequest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.requester_id, u.display_name, u.app_id, f.created_at
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.addressee_id = $1 AND f.status = 'pending' AND u.deleted_at IS NULL
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*friendship.PendingRequest
	for rows.Next() {
		p := &friendship.PendingRequest{}
		if err := rows.Scan(&p.FriendshipID, &p.RequesterID, &p.DisplayName, &p.AppID, &p.SentAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *FriendshipRepository) UpsertLabel(ctx context.Context, l *friendship.Label) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO friend_labels (friendship_id, user_id, tag, nickname, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (friendship_id, user_id) DO UPDATE
		SET tag = EXCLUDED.tag, nickname = EXCLUDED.nickname, updated_at = EXCLUDED.updated_at
	`, l.FriendshipID, l.UserID, l.Tag, l.Nickname, l.UpdatedAt)
	return err
}

func (r *FriendshipRepository) GetLabel(ctx context.Context, friendshipID, userID uuid.UUID) (*friendship.Label, error) {
	l := &friendship.Label{}
	err := r.pool.QueryRow(ctx, `
		SELECT friendship_id, user_id, tag, nickname, updated_at
		FROM friend_labels WHERE friendship_id = $1 AND user_id = $2
	`, friendshipID, userID).Scan(&l.FriendshipID, &l.UserID, &l.Tag, &l.Nickname, &l.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &friendship.Label{FriendshipID: friendshipID, UserID: userID}, nil
		}
		return nil, err
	}
	return l, nil
}
