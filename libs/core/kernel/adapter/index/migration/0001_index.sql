-- The schema of the index. A change of it is a new numbered file after this one.

CREATE TABLE chunks (
    id        INTEGER PRIMARY KEY,
    source_id INTEGER NOT NULL,
    vault_id  INTEGER NOT NULL,
    start     INTEGER NOT NULL,
    length    INTEGER NOT NULL,
    parent_id INTEGER REFERENCES chunks(id) ON DELETE CASCADE,
    location  TEXT,
    hash      TEXT NOT NULL,

    FOREIGN KEY (source_id, vault_id) REFERENCES sources (id, vault_id) ON DELETE CASCADE
);

CREATE VIRTUAL TABLE chunks_fts USING fts5 (
    text,
    content='',
    contentless_delete=1,
    prefix='2 3'
);

CREATE VIRTUAL TABLE chunks_vec USING vec0 (
    chunk_id integer primary key,
    vault_id integer,
    coarse   bit[1024]
);

CREATE TABLE coarse_recipe (
    id     INTEGER PRIMARY KEY CHECK (id = 1),
    recipe TEXT NOT NULL
);

CREATE TABLE headings (
    id       INTEGER PRIMARY KEY,
    note_id  INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    line     INTEGER NOT NULL,
    level    INTEGER NOT NULL,
    text     TEXT NOT NULL,

    UNIQUE (note_id, line)
);

CREATE VIRTUAL TABLE headings_fts USING fts5 (
    text,
    prefix='2 3'
);

CREATE TABLE links (
    note_id     INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    scheme      TEXT NOT NULL,
    target      TEXT NOT NULL,

    -- The last segment of the address under the same fold a note's name is
    -- held to, which is what the two are compared on: what [[notes/Entropy]],
    -- [[Entropy]] and [[entropy.MD]] have in common.
    folded_name TEXT NOT NULL,

    role        TEXT NOT NULL,
    type        TEXT,
    why         TEXT,
    label       TEXT, marker TEXT,

    PRIMARY KEY (note_id, position)
);

CREATE TABLE notes (
    source_id         INTEGER PRIMARY KEY,
    vault_id          INTEGER NOT NULL,

    -- The name a link written by name reaches this note by, as domain.FoldName
    -- computes it: one name whatever case and composition it is written in.
    folded_name       TEXT NOT NULL,

    title             TEXT NOT NULL,
    type              TEXT NOT NULL DEFAULT 'note',
    identifier        TEXT,
    frontmatter       TEXT,
    frontmatter_error TEXT,

    FOREIGN KEY (source_id, vault_id) REFERENCES sources (id, vault_id) ON DELETE CASCADE
);

CREATE TABLE open_tabs (
    vault    TEXT NOT NULL,
    position INTEGER NOT NULL,
    id       TEXT NOT NULL,
    kind     TEXT NOT NULL,
    path     TEXT NOT NULL,
    title    TEXT NOT NULL,
    state    TEXT NOT NULL,
    is_front INTEGER NOT NULL,
    PRIMARY KEY (vault, position)
);

CREATE TABLE problems (
    note_id INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    detail  TEXT NOT NULL
);

CREATE TABLE search_floors (
    vault  TEXT NOT NULL,
    recipe TEXT NOT NULL,
    floor  REAL NOT NULL,
    PRIMARY KEY (vault, recipe)
) WITHOUT ROWID;

CREATE VIRTUAL TABLE sections_fts USING fts5 (
    text,
    content='',
    contentless_delete=1,
    prefix='2 3'
);

CREATE TABLE sources (
    id                INTEGER PRIMARY KEY,
    vault_id          INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    path              TEXT NOT NULL,
    kind              TEXT NOT NULL,
    size              INTEGER NOT NULL,
    modified_at       INTEGER NOT NULL,
    content_hash      TEXT,
    recipe            TEXT,
    producer          TEXT,
    file_id           INTEGER,
    content_hash_time INTEGER,

    UNIQUE (vault_id, path)
);

CREATE TABLE text_versions (
    id                    INTEGER PRIMARY KEY,
    vault                 TEXT NOT NULL,
    path                  TEXT NOT NULL,
    plugin_id             TEXT NOT NULL,
    key                   TEXT NOT NULL,
    source_version_id     TEXT NOT NULL,
    source_property_key   TEXT NOT NULL DEFAULT '',
    source_property_value TEXT NOT NULL DEFAULT '',
    kind                  TEXT NOT NULL,
    file                  TEXT NOT NULL,
    input_hash            TEXT NOT NULL,
    text_hash             TEXT NOT NULL,
    create_time           INTEGER NOT NULL,
    UNIQUE (vault, path, plugin_id, key)
);

CREATE VIRTUAL TABLE titles_fts USING fts5 (
    text,
    prefix='2 3'
);

CREATE TABLE vaults (
    id         INTEGER PRIMARY KEY,
    identifier TEXT NOT NULL UNIQUE
);

CREATE TABLE vectors (
    hash      BLOB NOT NULL,
    recipe    TEXT NOT NULL,
    embedding BLOB NOT NULL
);

CREATE INDEX chunks_by_hash ON chunks (hash);

CREATE INDEX chunks_by_parent ON chunks (parent_id);

CREATE INDEX chunks_by_source ON chunks (source_id, vault_id);

CREATE INDEX chunks_by_vault ON chunks (vault_id, id);

CREATE INDEX chunks_by_vault_parent ON chunks (vault_id, parent_id, id);

CREATE INDEX links_by_name ON links (folded_name);

CREATE INDEX links_by_target ON links (scheme, target);

CREATE INDEX notes_by_folded_name ON notes (vault_id, folded_name);

CREATE INDEX notes_by_identifier ON notes (identifier);

CREATE INDEX notes_by_type ON notes (vault_id, type);

CREATE INDEX problems_by_note ON problems (note_id);

CREATE INDEX sources_by_fingerprint ON sources (vault_id, kind, path, size, modified_at);

CREATE UNIQUE INDEX sources_by_id_vault ON sources (id, vault_id);

CREATE INDEX sources_by_modified ON sources (vault_id, modified_at DESC, id DESC);

CREATE INDEX sources_by_producer ON sources (vault_id, kind) WHERE producer IS NOT NULL;

CREATE INDEX text_versions_by_plugin ON text_versions (vault, plugin_id);

CREATE UNIQUE INDEX vectors_by_hash ON vectors (hash, recipe);
