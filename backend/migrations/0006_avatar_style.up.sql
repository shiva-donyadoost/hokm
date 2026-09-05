-- DiceBear style alongside seed (ADR-0018). Empty => lorelei for legacy rows.
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_style TEXT;
