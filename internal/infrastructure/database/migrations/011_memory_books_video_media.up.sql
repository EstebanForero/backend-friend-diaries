ALTER TABLE diary_groups
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'everyday',
    ADD COLUMN IF NOT EXISTS emoji TEXT NOT NULL DEFAULT '📖',
    ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT 'rose';

ALTER TABLE note_images
    ADD COLUMN IF NOT EXISTS media_kind TEXT NOT NULL DEFAULT 'image',
    ADD COLUMN IF NOT EXISTS duration_seconds DOUBLE PRECISION;

ALTER TABLE note_images
    ADD CONSTRAINT note_images_media_kind_check
    CHECK (media_kind IN ('image', 'video'));

CREATE INDEX IF NOT EXISTS idx_note_images_media_kind ON note_images(media_kind);
CREATE INDEX IF NOT EXISTS idx_notes_group_recent
    ON notes(group_id, created_at DESC)
    WHERE is_deleted = FALSE;
