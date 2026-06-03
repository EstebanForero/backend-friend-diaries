CREATE TYPE reaction_type AS ENUM ('love', 'funny', 'sad', 'support', 'seen', 'favorite');

CREATE TABLE reactions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id       UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reaction_type reaction_type NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (note_id, user_id)
);

CREATE INDEX idx_reactions_note ON reactions (note_id);
