DROP INDEX IF EXISTS idx_notes_author_client_id;
ALTER TABLE notes DROP COLUMN IF EXISTS client_id;
