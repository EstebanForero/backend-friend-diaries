package repository

import (
	"context"
	"encoding/json"

	"github.com/friend-diaries/backend/internal/domain/notification"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Create(ctx context.Context, n *notification.Notification) error {
	metadata, err := json.Marshal(n.Metadata)
	if err != nil {
		metadata = []byte("{}")
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO notifications (id, recipient_id, actor_id, type, reference_id, ref_type, metadata, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, n.ID, n.RecipientID, n.ActorID, n.Type, n.ReferenceID, n.RefType, metadata, n.IsRead, n.CreatedAt)
	return err
}

func (r *NotificationRepository) FindByRecipient(ctx context.Context, recipientID uuid.UUID, limit, offset int) ([]*notification.Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.recipient_id, n.actor_id,
		       COALESCE(u.display_name, 'Deleted User') AS actor_display_name,
		       n.type, n.reference_id, n.ref_type, n.metadata, n.is_read, n.created_at
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		WHERE n.recipient_id = $1
		ORDER BY n.created_at DESC
		LIMIT $2 OFFSET $3
	`, recipientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*notification.Notification
	for rows.Next() {
		n := &notification.Notification{}
		var metadata []byte
		if err := rows.Scan(&n.ID, &n.RecipientID, &n.ActorID, &n.ActorDisplayName,
			&n.Type, &n.ReferenceID, &n.RefType, &metadata, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		if len(metadata) > 0 {
			_ = json.Unmarshal(metadata, &n.Metadata)
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepository) CountUnread(ctx context.Context, recipientID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE recipient_id=$1 AND is_read=FALSE
	`, recipientID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, recipientID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read=TRUE WHERE id=$1 AND recipient_id=$2
	`, id, recipientID)
	return err
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, recipientID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read=TRUE WHERE recipient_id=$1 AND is_read=FALSE
	`, recipientID)
	return err
}

func (r *NotificationRepository) DeleteByRecipientReferenceType(ctx context.Context, recipientID, referenceID uuid.UUID, notificationType notification.Type) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM notifications
		WHERE recipient_id=$1 AND reference_id=$2 AND type=$3
	`, recipientID, referenceID, notificationType)
	return err
}
