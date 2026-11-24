-- The files a vault lists and does not hold. Derived data: a plugin sends it again after a rebuild.
-- mtime is nanoseconds since the epoch.
CREATE TABLE absent_files (
    vault_id TEXT    NOT NULL,
    path     TEXT    NOT NULL,
    size     INTEGER NOT NULL,
    mtime    INTEGER NOT NULL,
    hash     TEXT    NOT NULL,
    provider TEXT    NOT NULL,
    PRIMARY KEY (vault_id, path)
) WITHOUT ROWID;
