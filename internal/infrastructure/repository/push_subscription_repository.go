package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PushSubscription struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Endpoint  string
	P256dh    string
	Auth      string
	CreatedAt time.Time
}

type PushSubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewPushSubscriptionRepository(pool *pgxpool.Pool) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{pool: pool}
}

func (r *PushSubscriptionRepository) Save(ctx context.Context, sub *PushSubscription) error {
	sub.ID = uuid.New()
	sub.CreatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (endpoint) DO UPDATE SET user_id=$2, p256dh=$4, auth=$5
	`, sub.ID, sub.UserID, sub.Endpoint, sub.P256dh, sub.Auth, sub.CreatedAt)
	return err
}

func (r *PushSubscriptionRepository) Delete(ctx context.Context, endpoint string, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM push_subscriptions WHERE endpoint=$1 AND user_id=$2
	`, endpoint, userID)
	return err
}

func (r *PushSubscriptionRepository) ListByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]PushSubscription, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, endpoint, p256dh, auth, created_at
		FROM push_subscriptions WHERE user_id = ANY($1)
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
