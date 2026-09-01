ALTER TABLE notes ADD COLUMN client_id UUID;

CREATE UNIQUE INDEX idx_notes_author_client_id
    ON notes (author_id, client_id)
    WHERE client_id IS NOT NULL;
