-- A note may carry an identifier and links. Both are read from the file, and
-- neither is invented by the index: what is stored here is what was written.
ALTER TABLE notes ADD COLUMN note_id TEXT;

-- The name a note is found by when a link is written by name (ADR-0011). Stored
-- rather than computed, because resolution asks for it on every link.
ALTER TABLE notes ADD COLUMN basename TEXT NOT NULL DEFAULT '';

CREATE INDEX notes_by_note_id ON notes (vault_id, note_id);
CREATE INDEX notes_by_basename ON notes (vault_id, basename);

-- One row per link as it was written. Where it points is a question asked of
-- this table joined with notes, not a fact stored in it: adding a file can
-- resolve a link that was dangling, and deleting one can break a link that
-- worked.
CREATE TABLE links (
    vault_id  TEXT NOT NULL,
    from_path TEXT NOT NULL,
    scheme    TEXT NOT NULL,
    value     TEXT NOT NULL,
    role      TEXT NOT NULL,
    type      TEXT,
    note      TEXT,
    label     TEXT,
    position  INTEGER NOT NULL,
    PRIMARY KEY (vault_id, from_path, position),
    FOREIGN KEY (vault_id, from_path) REFERENCES notes(vault_id, path) ON DELETE CASCADE
);

CREATE INDEX links_by_target ON links (vault_id, scheme, value);

-- What could not be repaired and is worth showing: a link with no role, an
-- unreadable frontmatter block. Kept beside the note so that a "vault problems"
-- view is a query rather than a rescan.
CREATE TABLE problems (
    vault_id TEXT NOT NULL,
    path     TEXT NOT NULL,
    detail   TEXT NOT NULL,
    FOREIGN KEY (vault_id, path) REFERENCES notes(vault_id, path) ON DELETE CASCADE
);

CREATE INDEX problems_by_vault ON problems (vault_id);
