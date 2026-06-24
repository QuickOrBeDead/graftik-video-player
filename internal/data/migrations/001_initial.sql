CREATE TABLE IF NOT EXISTS playlists (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    shuffle INTEGER NOT NULL DEFAULT 0,
    repeat INTEGER NOT NULL DEFAULT 0,
    current_item TEXT REFERENCES playlist_items(id) ON DELETE SET NULL,
    current_volume REAL
);

CREATE TABLE IF NOT EXISTS playlist_items (
    id TEXT PRIMARY KEY,
    playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    title TEXT NOT NULL,
    elapsed_time REAL NOT NULL DEFAULT 0,
    is_playing INTEGER NOT NULL DEFAULT 0,
    duration REAL,
    progress_percent REAL NOT NULL DEFAULT 0,
    last_watched INTEGER,
    order_index REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_playlist_items_playlist_id_order
    ON playlist_items(playlist_id, order_index);

PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA journal_size_limit = 10485760;
