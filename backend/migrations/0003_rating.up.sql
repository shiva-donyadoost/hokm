-- Rating support for Phase 13.
ALTER TABLE statistics ADD COLUMN IF NOT EXISTS rating INT NOT NULL DEFAULT 1000;
