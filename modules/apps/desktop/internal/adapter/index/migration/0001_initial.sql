CREATE TABLE vaults (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    path TEXT NOT NULL
);

-- vault_id is part of every primary key rather than a column to be remembered:
-- a query that forgets it returns another vault's notes, silently, and no test
-- with a single vault can see that.
CREATE TABLE files (
    vault_id TEXT    NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    path     TEXT    NOT NULL,
    size     INTEGER NOT NULL,
    mtime    INTEGER NOT NULL,
    PRIMARY KEY (vault_id, path)
);

CREATE TABLE notes (
    vault_id        TEXT NOT NULL,
    path            TEXT NOT NULL,
    title           TEXT NOT NULL,
    frontmatter     TEXT,
    frontmatter_err TEXT,
    PRIMARY KEY (vault_id, path),
    FOREIGN KEY (vault_id, path) REFERENCES files(vault_id, path) ON DELETE CASCADE
);

CREATE TABLE headings (
    vault_id TEXT    NOT NULL,
    path     TEXT    NOT NULL,
    level    INTEGER NOT NULL,
    text     TEXT    NOT NULL,
    pos      INTEGER NOT NULL
);

CREATE INDEX headings_by_note ON headings (vault_id, path);

-- vault_id and path are stored but not tokenised: they scope and locate a hit
-- rather than being something to match against.
CREATE VIRTUAL TABLE notes_fts USING fts5 (
    title,
    body,
    vault_id UNINDEXED,
    path     UNINDEXED
);
