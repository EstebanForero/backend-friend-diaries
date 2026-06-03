CREATE TABLE friendships (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (requester_id, addressee_id),
    CHECK (requester_id <> addressee_id)
);

CREATE TABLE friend_labels (
    friendship_id UUID NOT NULL REFERENCES friendships(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag           VARCHAR(50) NOT NULL DEFAULT '',
    nickname      VARCHAR(100) NOT NULL DEFAULT '',
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (friendship_id, user_id)
);

CREATE INDEX idx_friendships_requester ON friendships (requester_id);
CREATE INDEX idx_friendships_addressee ON friendships (addressee_id);
CREATE INDEX idx_friendships_status    ON friendships (status);
