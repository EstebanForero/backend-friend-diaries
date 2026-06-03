package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/friend-diaries/backend/internal/domain/note"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NoteRepository struct {
	pool *pgxpool.Pool
}

func NewNoteRepository(pool *pgxpool.Pool) *NoteRepository {
	return &NoteRepository{pool: pool}
}

func (r *NoteRepository) Create(ctx context.Context, n *note.Note) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notes (id, group_id, author_id, diary_date, content, is_edited, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, n.ID, n.GroupID, n.AuthorID, n.DiaryDate, n.Content, n.IsEdited, n.IsDeleted, n.CreatedAt, n.UpdatedAt)
	return err
}

func (r *NoteRepository) FindByID(ctx context.Context, id uuid.UUID) (*note.Note, error) {
	n := &note.Note{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, group_id, author_id, diary_date, content, is_edited, is_deleted, created_at, updated_at, deleted_at
		FROM notes WHERE id=$1
	`, id).Scan(&n.ID, &n.GroupID, &n.AuthorID, &n.DiaryDate, &n.Content,
		&n.IsEdited, &n.IsDeleted, &n.CreatedAt, &n.UpdatedAt, &n.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("note not found")
		}
		return nil, err
	}
	return n, nil
}

func (r *NoteRepository) FindByGroupAndDate(ctx context.Context, groupID uuid.UUID, date time.Time) ([]*note.Note, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, group_id, author_id, diary_date, content, is_edited, is_deleted, created_at, updated_at, deleted_at
		FROM notes WHERE group_id=$1 AND diary_date=$2
		ORDER BY created_at ASC
	`, groupID, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*note.Note
	for rows.Next() {
		n := &note.Note{}
		if err := rows.Scan(&n.ID, &n.GroupID, &n.AuthorID, &n.DiaryDate, &n.Content,
			&n.IsEdited, &n.IsDeleted, &n.CreatedAt, &n.UpdatedAt, &n.DeletedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (r *NoteRepository) Update(ctx context.Context, n *note.Note) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notes SET content=$1, is_edited=$2, updated_at=$3 WHERE id=$4
	`, n.Content, n.IsEdited, n.UpdatedAt, n.ID)
	return err
}

func (r *NoteRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE notes SET is_deleted=TRUE, deleted_at=$1, updated_at=$1 WHERE id=$2
	`, now, id)
	return err
}

func (r *NoteRepository) CreateShareToken(ctx context.Context, noteID uuid.UUID, token string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO note_share_tokens (note_id, token, expires_at) VALUES ($1, $2, $3)
	`, noteID, token, expiresAt)
	return err
}

func (r *NoteRepository) CalendarSummary(ctx context.Context, groupID uuid.UUID, month string, authorID *uuid.UUID) (map[string]int, error) {
	// Parse month as first day of that month.
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, fmt.Errorf("invalid month format: %w", err)
	}
	end := start.AddDate(0, 1, 0)

	rows, err := r.pool.Query(ctx, `
		SELECT diary_date::text, COUNT(*)::int
		FROM notes
		WHERE group_id = $1
		  AND diary_date >= $2
		  AND diary_date < $3
		  AND is_deleted = false
		  AND ($4::uuid IS NULL OR author_id = $4)
		GROUP BY diary_date
		ORDER BY diary_date
	`, groupID, start, end, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var dateStr string
		var count int
		if err := rows.Scan(&dateStr, &count); err != nil {
			return nil, err
		}
		result[dateStr] = count
	}
	return result, rows.Err()
}

func (r *NoteRepository) FindByShareToken(ctx context.Context, token string) (*note.Note, error) {
	n := &note.Note{}
	err := r.pool.QueryRow(ctx, `
		SELECT n.id, n.group_id, n.author_id, n.diary_date, n.content, n.is_edited, n.is_deleted,
		       n.created_at, n.updated_at, n.deleted_at
		FROM notes n
		JOIN note_share_tokens st ON st.note_id = n.id
		WHERE st.token=$1 AND st.expires_at > NOW()
	`, token).Scan(&n.ID, &n.GroupID, &n.AuthorID, &n.DiaryDate, &n.Content,
		&n.IsEdited, &n.IsDeleted, &n.CreatedAt, &n.UpdatedAt, &n.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("share token not found or expired")
		}
		return nil, err
	}
	return n, nil
}
