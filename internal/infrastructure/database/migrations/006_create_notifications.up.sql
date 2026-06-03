CREATE TYPE notification_type AS ENUM (
    'new_note',
    'reaction',
    'reply',
    'group_invitation',
    'invitation_accepted'
);

CREATE TYPE ref_type AS ENUM ('note', 'group', 'reply');

CREATE TABLE notifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type         notification_type NOT NULL,
    reference_id UUID NOT NULL,
    ref_type     ref_type NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}',
    is_read      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient ON notifications (recipient_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications (recipient_id) WHERE is_read = FALSE;
