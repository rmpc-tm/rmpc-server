ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_player_id_key;
CREATE INDEX IF NOT EXISTS idx_sessions_player_id ON sessions(player_id);
