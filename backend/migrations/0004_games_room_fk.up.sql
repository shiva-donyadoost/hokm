-- Rooms are in-process (not persisted). The games.room_id FK blocked every
-- ApplyMatch write, so statistics stayed at zero (ADR-0012).
ALTER TABLE games DROP CONSTRAINT IF EXISTS games_room_id_fkey;
