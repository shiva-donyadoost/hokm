-- User-selectable DiceBear seeds (ADR-0017).
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_seed TEXT;
