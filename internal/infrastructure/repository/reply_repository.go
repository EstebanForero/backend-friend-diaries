package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/friend-diaries/backend/internal/domain/reply"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReplyRepository struct {
	pool *pgxpool.Pool
}

func NewReplyRepository(pool *pgxpool.Pool) *ReplyRepository {
	return &ReplyRepository{pool: pool}
}

func (r *ReplyRepository) Create(ctx context.Context, rep *reply.Reply) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO replies (id, note_id, author_id, content, is_edited, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, rep.ID, rep.NoteID, rep.AuthorID, rep.Content, rep.IsEdited, rep.IsDeleted, rep.CreatedAt, rep.UpdatedAt)
	return err
}

func (r *ReplyRepository) FindByID(ctx context.Context, id uuid.UUID) (*reply.Reply, error) {
	rep := &reply.Reply{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, author_id, content, is_edited, is_deleted, created_at, updated_at, deleted_at
		FROM replies WHERE id=$1
	`, id).Scan(&rep.ID, &rep.NoteID, &rep.AuthorID, &rep.Content, &rep.IsEdited,
		&rep.IsDeleted, &rep.CreatedAt, &rep.UpdatedAt, &rep.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reply not found")
		}
		return nil, err
	}
	return rep, nil
}

func (r *ReplyRepository) FindByNote(ctx context.Context, noteID uuid.UUID) ([]*reply.Reply, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, author_id, content, is_edited, is_deleted, created_at, updated_at, deleted_at
		FROM replies WHERE note_id=$1 AND is_deleted=FALSE
		ORDER BY created_at ASC
	`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []*reply.Reply
	for rows.Next() {
		rep := &reply.Reply{}
		if err := rows.Scan(&rep.ID, &rep.NoteID, &rep.AuthorID, &rep.Content, &rep.IsEdited,
			&rep.IsDeleted, &rep.CreatedAt, &rep.UpdatedAt, &rep.DeletedAt); err != nil {
			return nil, err
		}
		replies = append(replies, rep)
	}
	return replies, rows.Err()
}

func (r *ReplyRepository) Update(ctx context.Context, rep *reply.Reply) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE replies SET content=$1, is_edited=$2, updated_at=$3 WHERE id=$4
	`, rep.Content, rep.IsEdited, rep.UpdatedAt, rep.ID)
	return err
}

func (r *ReplyRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE replies SET is_deleted=TRUE, deleted_at=$1, updated_at=$1 WHERE id=$2
	`, now, id)
	return err
}
