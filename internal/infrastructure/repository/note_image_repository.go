package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NoteImage struct {
	ID         uuid.UUID
	NoteID     *uuid.UUID
	UploaderID uuid.UUID
	ObjectKey  string
	MimeType   string
	SizeBytes  int64
	CreatedAt  time.Time
}

type NoteImageRepository struct {
	pool *pgxpool.Pool
}

func NewNoteImageRepository(pool *pgxpool.Pool) *NoteImageRepository {
	return &NoteImageRepository{pool: pool}
}

func (r *NoteImageRepository) Create(ctx context.Context, img *NoteImage) error {
	img.ID = uuid.New()
	img.CreatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO note_images (id, note_id, uploader_id, object_key, mime_type, size_bytes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, img.ID, img.NoteID, img.UploaderID, img.ObjectKey, img.MimeType, img.SizeBytes, img.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting note_image: %w", err)
	}
	return nil
}

func (r *NoteImageRepository) AttachToNote(ctx context.Context, imageIDs []uuid.UUID, noteID uuid.UUID, uploaderID uuid.UUID) error {
	for _, id := range imageIDs {
		_, err := r.pool.Exec(ctx, `
			UPDATE note_images SET note_id = $1
			WHERE id = $2 AND uploader_id = $3 AND note_id IS NULL
		`, noteID, id, uploaderID)
		if err != nil {
			return fmt.Errorf("attaching image %s: %w", id, err)
		}
	}
	return nil
}

func (r *NoteImageRepository) ListByNoteID(ctx context.Context, noteID uuid.UUID) ([]NoteImage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, uploader_id, object_key, mime_type, size_bytes, created_at
		FROM note_images WHERE note_id = $1 ORDER BY created_at ASC
	`, noteID)
	if err != nil {
		return nil, fmt.Errorf("querying note_images: %w", err)
	}
	defer rows.Close()

	var images []NoteImage
	for rows.Next() {
		var img NoteImage
		if err := rows.Scan(&img.ID, &img.NoteID, &img.UploaderID, &img.ObjectKey,
			&img.MimeType, &img.SizeBytes, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning note_image: %w", err)
		}
		images = append(images, img)
	}
	return images, nil
}

func (r *NoteImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*NoteImage, error) {
	var img NoteImage
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, uploader_id, object_key, mime_type, size_bytes, created_at
		FROM note_images WHERE id = $1
	`, id).Scan(&img.ID, &img.NoteID, &img.UploaderID, &img.ObjectKey,
		&img.MimeType, &img.SizeBytes, &img.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("finding note_image: %w", err)
	}
	return &img, nil
}

func (r *NoteImageRepository) Delete(ctx context.Context, id uuid.UUID, uploaderID uuid.UUID) (string, error) {
	var objectKey string
	err := r.pool.QueryRow(ctx, `
		DELETE FROM note_images WHERE id = $1 AND uploader_id = $2 RETURNING object_key
	`, id, uploaderID).Scan(&objectKey)
	if err != nil {
		return "", fmt.Errorf("deleting note_image: %w", err)
	}
	return objectKey, nil
}
