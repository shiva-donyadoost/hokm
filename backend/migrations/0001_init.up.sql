-- Core identity + persistence schema (Phase 11).
-- IDs are application-generated text (crypto random hex), not DB-generated,
-- so the same identifiers flow through the engine and transport layers.

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_guest      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens (user_id);

CREATE TABLE IF NOT EXISTS rooms (
    id          TEXT PRIMARY KEY,
    code        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    visibility  TEXT NOT NULL CHECK (visibility IN ('public', 'private')),
    host_id     TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'lobby' CHECK (status IN ('lobby', 'in_game', 'finished')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at   TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS games (
    id           TEXT PRIMARY KEY,
    room_id      TEXT NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    rounds_to_win INT NOT NULL,
    winner_team  INT CHECK (winner_team IN (0, 1)),
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS game_players (
    game_id       TEXT NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    seat          INT NOT NULL CHECK (seat BETWEEN 0 AND 3),
    user_id       TEXT NOT NULL,
    username      TEXT NOT NULL,
    is_ai         BOOLEAN NOT NULL DEFAULT FALSE,
    ai_difficulty TEXT,
    team          INT NOT NULL CHECK (team IN (0, 1)),
    PRIMARY KEY (game_id, seat)
);

CREATE INDEX IF NOT EXISTS idx_game_players_user ON game_players (user_id);

CREATE TABLE IF NOT EXISTS game_events (
    game_id    TEXT NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    seq        INT NOT NULL,
    kind       TEXT NOT NULL,
    payload    JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (game_id, seq)
);

CREATE TABLE IF NOT EXISTS statistics (
    user_id       TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    games_played  INT NOT NULL DEFAULT 0,
    wins          INT NOT NULL DEFAULT 0,
    losses        INT NOT NULL DEFAULT 0,
    rounds_won    INT NOT NULL DEFAULT 0,
    rounds_lost   INT NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
