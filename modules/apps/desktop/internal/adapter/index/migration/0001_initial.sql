-- A vault the index knows about.
--
-- `identifier` is the ULID in the vault's own config file, which travels with
-- the folder. `id` is what this database calls the vault, and is what every
-- other row here carries.
CREATE TABLE vaults (
    id         INTEGER PRIMARY KEY,
    identifier TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    path       TEXT NOT NULL
);

-- A note, and what a scan knows about the file it came from.
--
-- `id` is the note's identity inside this database, and what every other table
-- points at.
--
-- `size` and `modified_at` are what let the next scan skip the file without
-- opening it.
--
-- `identifier` is the ULID written in the file, when there is one. Most notes
-- have none, and a copied file carries a copy of it.
CREATE TABLE notes (
    id                INTEGER PRIMARY KEY,
    vault_id          INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    path              TEXT NOT NULL,

    -- The name a note is found by when a link is written by name. NOCASE on the
    -- column and not on the comparison, so that [[entropy]] finds Entropy.md
    -- through the index below.
    basename          TEXT NOT NULL COLLATE NOCASE,

    title             TEXT NOT NULL,
    identifier        TEXT,
    frontmatter       TEXT,
    frontmatter_error TEXT,
    size              INTEGER NOT NULL,
    modified_at       INTEGER NOT NULL,

    UNIQUE (vault_id, path)
);

-- Carries the path so that resolution is answered from the index alone.
CREATE INDEX notes_by_basename ON notes (vault_id, basename, path);

-- Every scan asks one question of every note in the vault: has this file
-- changed. This answers it without reading the notes themselves.
CREATE INDEX notes_by_fingerprint ON notes (vault_id, path, size, modified_at);

-- An identifier names one note in the world, and is looked up without a vault.
CREATE INDEX notes_by_identifier ON notes (identifier);

CREATE TABLE headings (
    note_id  INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    level    INTEGER NOT NULL,
    text     TEXT NOT NULL,
    PRIMARY KEY (note_id, position)
);

-- The search index, and nothing else: `content=''` keeps no copy of what was
-- indexed. A result is a title and a path, and the text is on disk.
--
-- `contentless_delete` is what makes a row replaceable without that copy, which
-- a rescan of an edited note needs.
--
-- The rowid is the note's own id. It is the only key this kind of table has.
CREATE VIRTUAL TABLE notes_fts USING fts5 (
    title,
    body,
    content='',
    contentless_delete=1
);
