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

-- A file the index has read, and what a scan knows about it. `kind` says what
-- sort of file it is. What only one kind has is stored in a table of its own.
--
-- `id` is the source's identity inside this database, and what every other
-- table points at.
--
-- `size` and `modified_at` are what let the next scan skip the file without
-- opening it, for every kind.
--
-- `hash` is the address the content itself gives the source, and is null until
-- something computes it. A source is saved and removed by its path, and a hash
-- is what lets a moved file keep what was derived from it.
--
-- `recipe` is what extracted the text, and is null while nothing has.
CREATE TABLE sources (
    id          INTEGER PRIMARY KEY,
    vault_id    INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    path        TEXT NOT NULL,
    kind        TEXT NOT NULL,
    size        INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    hash        TEXT,
    recipe      TEXT,

    UNIQUE (vault_id, path)
);

-- Every row filed under a vault carries the vault, and names the pair as its
-- foreign key, so a row cannot claim a vault its source does not belong to.
CREATE UNIQUE INDEX sources_by_vault ON sources (id, vault_id);

-- Every scan asks one question of every source of one kind: has this file
-- changed. This answers it without reading the files themselves.
CREATE INDEX sources_by_fingerprint ON sources (vault_id, kind, path, size, modified_at);

-- What only a note has: the names a link reaches it by, and the frontmatter
-- they are written in. A note's row number is its source's, so a question that
-- needs only the file joins `sources` on it.
--
-- `identifier` is the ULID written in the file, when there is one. Most notes
-- have none, and a copied file carries a copy of it.
CREATE TABLE notes (
    source_id         INTEGER PRIMARY KEY,
    vault_id          INTEGER NOT NULL,

    -- The name a note is found by when a link is written by name. NOCASE on the
    -- column and not on the comparison, so that [[entropy]] finds Entropy.md
    -- through the index below.
    basename          TEXT NOT NULL COLLATE NOCASE,

    title             TEXT NOT NULL,
    identifier        TEXT,
    frontmatter       TEXT,
    frontmatter_error TEXT,

    FOREIGN KEY (source_id, vault_id) REFERENCES sources (id, vault_id) ON DELETE CASCADE
);

-- A name means something inside one vault, so the vault leads. The path a
-- candidate is reported by comes from the source, which is one lookup by row
-- number away.
CREATE INDEX notes_by_basename ON notes (vault_id, basename);

-- An identifier names one note in the world, and is looked up without a vault.
CREATE INDEX notes_by_identifier ON notes (identifier);

CREATE TABLE headings (
    note_id  INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    line     INTEGER NOT NULL,
    level    INTEGER NOT NULL,
    text     TEXT NOT NULL,
    PRIMARY KEY (note_id, line)
);
