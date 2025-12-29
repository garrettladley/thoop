CREATE TABLE IF NOT EXISTS sync_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    backfill_complete INTEGER NOT NULL DEFAULT 0,
    backfill_watermark DATETIME,
    last_full_sync DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO sync_state (id) VALUES (1);

CREATE TABLE IF NOT EXISTS cycles (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    start DATETIME NOT NULL,
    end DATETIME,
    timezone_offset TEXT NOT NULL,
    score_state TEXT NOT NULL,
    score_json TEXT,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_cycles_start ON cycles(start DESC);

CREATE TABLE IF NOT EXISTS recoveries (
    cycle_id INTEGER PRIMARY KEY,
    sleep_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    score_state TEXT NOT NULL,
    score_json TEXT,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sleeps (
    id TEXT PRIMARY KEY,
    cycle_id INTEGER NOT NULL,
    v1_id INTEGER,
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    start DATETIME NOT NULL,
    end DATETIME NOT NULL,
    timezone_offset TEXT NOT NULL,
    nap INTEGER NOT NULL DEFAULT 0,
    score_state TEXT NOT NULL,
    score_json TEXT,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_sleeps_cycle_id ON sleeps(cycle_id);

CREATE TABLE IF NOT EXISTS workouts (
    id TEXT PRIMARY KEY,
    v1_id INTEGER,
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    start DATETIME NOT NULL,
    end DATETIME NOT NULL,
    timezone_offset TEXT NOT NULL,
    sport_name TEXT NOT NULL,
    score_state TEXT NOT NULL,
    score_json TEXT,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_workouts_start ON workouts(start DESC);
