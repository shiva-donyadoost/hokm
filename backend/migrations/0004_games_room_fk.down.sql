ALTER TABLE games
    ADD CONSTRAINT games_room_id_fkey
    FOREIGN KEY (room_id) REFERENCES rooms (id) ON DELETE CASCADE;
