CREATE TABLE IF NOT EXISTS tokens (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type TEXT NOT NULL,
    expiry DATETIME NOT NULL
);
