package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/friend-diaries/backend/internal/domain/reaction"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReactionRepository struct {
	pool *pgxpool.Pool
}

func NewReactionRepository(pool *pgxpool.Pool) *ReactionRepository {
	return &ReactionRepository{pool: pool}
}

func (r *ReactionRepository) Upsert(ctx context.Context, react *reaction.Reaction) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reactions (id, note_id, user_id, reaction_type, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (note_id, user_id) DO UPDATE SET reaction_type=EXCLUDED.reaction_type, id=EXCLUDED.id
	`, react.ID, react.NoteID, react.UserID, react.ReactionType, react.CreatedAt)
	return err
}

func (r *ReactionRepository) Delete(ctx context.Context, noteID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM reactions WHERE note_id=$1 AND user_id=$2`, noteID, userID)
	return err
}

func (r *ReactionRepository) FindByNote(ctx context.Context, noteID uuid.UUID) ([]*reaction.Reaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, user_id, reaction_type, created_at FROM reactions WHERE note_id=$1
	`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []*reaction.Reaction
	for rows.Next() {
		react := &reaction.Reaction{}
		if err := rows.Scan(&react.ID, &react.NoteID, &react.UserID, &react.ReactionType, &react.CreatedAt); err != nil {
			return nil, err
		}
		reactions = append(reactions, react)
	}
	return reactions, rows.Err()
}

func (r *ReactionRepository) FindByNoteAndUser(ctx context.Context, noteID, userID uuid.UUID) (*reaction.Reaction, error) {
	react := &reaction.Reaction{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, user_id, reaction_type, created_at FROM reactions WHERE note_id=$1 AND user_id=$2
	`, noteID, userID).Scan(&react.ID, &react.NoteID, &react.UserID, &react.ReactionType, &react.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reaction not found")
		}
		return nil, err
	}
	return react, nil
}
