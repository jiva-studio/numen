-- The file change feed of each vault: the latest event of each kind of each source, numbered in the
-- order the index stored them. change_time is when the event was stored, which a removal is dropped by.

CREATE TABLE change_feeds (
    vault_id    INTEGER PRIMARY KEY REFERENCES vaults(id) ON DELETE CASCADE,
    feed_id     TEXT NOT NULL,
    last_seq    INTEGER NOT NULL DEFAULT 0,
    trimmed_seq INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE file_changes (
    vault_id    INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    source_id   INTEGER NOT NULL,
    event       TEXT NOT NULL,
    seq         INTEGER NOT NULL,
    change_time INTEGER NOT NULL,

    PRIMARY KEY (vault_id, source_id, event)
) WITHOUT ROWID;

CREATE INDEX file_changes_by_seq ON file_changes (vault_id, seq);

INSERT INTO change_feeds (vault_id, feed_id) SELECT id, lower(hex(randomblob(16))) FROM vaults;
