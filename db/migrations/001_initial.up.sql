-- Game mode enum
CREATE TYPE game_mode AS ENUM ('author', 'gold', 'custom');

-- Players table
CREATE TABLE players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    openplanet_id VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sessions table (long-lived tokens)
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_sessions_player_id ON sessions(player_id);

-- Banned players
CREATE TABLE banned_players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID UNIQUE NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    reason TEXT,
    banned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Scores table (RMPC runs)
CREATE TABLE scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    game_mode game_mode NOT NULL,
    score INTEGER NOT NULL,
    maps_completed INTEGER NOT NULL,
    maps_skipped INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_scores_game_mode ON scores(game_mode);
CREATE INDEX idx_scores_created_at ON scores(created_at);
CREATE INDEX idx_scores_player_id ON scores(player_id);
CREATE INDEX idx_scores_score ON scores(score DESC);

-- Metrics table (simple counters by day)
CREATE TABLE metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    UNIQUE(name, date)
);

CREATE INDEX idx_metrics_name_date ON metrics(name, date);
