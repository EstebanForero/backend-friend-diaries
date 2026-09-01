DROP INDEX IF EXISTS idx_notes_group_recent;
DROP INDEX IF EXISTS idx_note_images_media_kind;
ALTER TABLE note_images DROP CONSTRAINT IF EXISTS note_images_media_kind_check;
ALTER TABLE note_images
    DROP COLUMN IF EXISTS duration_seconds,
    DROP COLUMN IF EXISTS media_kind;
ALTER TABLE diary_groups
    DROP COLUMN IF EXISTS color,
    DROP COLUMN IF EXISTS emoji,
    DROP COLUMN IF EXISTS kind;
