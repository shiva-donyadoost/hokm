-- Chat and social schema (Phase 12 tables created with the core migration
-- set so Phase 12 needs no schema churn).
CREATE TABLE IF NOT EXISTS chat_messages (
    id         BIGSERIAL PRIMARY KEY,
    room_id    TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    username   TEXT NOT NULL,
    body       TEXT NOT NULL CHECK (length(body) BETWEEN 1 AND 500),
    is_system  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_room ON chat_messages (room_id, id);
