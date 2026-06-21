-- Make all existing users premium and set premium as the default
ALTER TABLE users ALTER COLUMN is_premium SET DEFAULT true;
UPDATE users SET is_premium = true WHERE NOT is_premium;

-- Track when a user accepted the data treatment policy
ALTER TABLE users ADD COLUMN IF NOT EXISTS consent_given_at TIMESTAMPTZ;

-- Images attached to notes
CREATE TABLE IF NOT EXISTS note_images (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id      UUID        REFERENCES notes(id) ON DELETE CASCADE,
    uploader_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_key   TEXT        NOT NULL UNIQUE,
    mime_type    TEXT        NOT NULL DEFAULT 'image/jpeg',
    size_bytes   BIGINT      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_note_images_note_id ON note_images(note_id);
