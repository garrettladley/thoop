-- move secrets to OS keyring, keep only metadata in SQLite
-- all users will be logged out in favor of a more secure solution

-- step 1: create new table with only metadata columns
CREATE TABLE tokens_new (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    token_type TEXT NOT NULL DEFAULT 'Bearer',
    expiry DATETIME NOT NULL
);

-- step 2: migrate non-sensitive data (if any row exists)
INSERT OR IGNORE INTO tokens_new (id, token_type, expiry)
SELECT id, token_type, expiry FROM tokens WHERE id = 1;

-- step 3: drop old table
DROP TABLE tokens;

-- step 4: rename new table
ALTER TABLE tokens_new RENAME TO tokens;
