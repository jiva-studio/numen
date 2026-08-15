-- A note may carry an identifier and links. Both are read from the file, and
-- neither is invented by the index: what is stored here is what was written.
ALTER TABLE notes ADD COLUMN note_id TEXT;

-- The name a note is found by when a link is written by name (ADR-0011). Stored
-- rather than computed, because resolution asks for it on every link.
--
-- NOCASE on the column rather than on the comparison: people type [[entropy]]
-- for Entropy.md, and folding case in the query instead would make the index
-- below unusable and turn every link into a scan of the vault.
ALTER TABLE notes ADD COLUMN basename TEXT NOT NULL DEFAULT '' COLLATE NOCASE;

-- Covering: resolution asks for the path, and an index that does not carry it
-- sends SQLite back to the table, which it avoids by reading the table instead.
CREATE INDEX notes_by_basename ON notes (vault_id, basename, path);

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

    -- The last segment of the address, without an extension: what a link
    -- written as [[notes/Entropy]] and one written as [[Entropy]] have in
    -- common. Stored so that the backwards question — who points at this note —
    -- is an index lookup rather than a pattern match over every link.
    value_base TEXT NOT NULL DEFAULT '' COLLATE NOCASE,

    position  INTEGER NOT NULL,
    PRIMARY KEY (vault_id, from_path, position),
    FOREIGN KEY (vault_id, from_path) REFERENCES notes(vault_id, path) ON DELETE CASCADE
);

-- Both carry enough to find the row again without reading every link in the
-- vault: the backwards question is asked once per note opened.
CREATE INDEX links_by_target ON links (vault_id, scheme, value);
CREATE INDEX links_by_name ON links (vault_id, value_base);

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

-- Links are not derivable from what the previous schema stored, so this
-- migration cannot fill them in: it empties what it changed and lets the next
-- scan refill it (ADR-0015). Without this, an index that already exists keeps
-- every note as "unchanged" on size and modification time, and answers no link
-- question until the user happens to edit each file.
--
-- The vaults themselves are left alone: they are what the rows point at, and
-- they are known from the registry rather than from a file.
DELETE FROM notes_fts;
DELETE FROM headings;
DELETE FROM links;
DELETE FROM problems;
DELETE FROM notes;
DELETE FROM files;

-- An identifier names one note in the world, so it is looked up without a
-- vault (ADR-0011). An index led by vault_id cannot serve that.
CREATE INDEX notes_by_id ON notes (note_id);
