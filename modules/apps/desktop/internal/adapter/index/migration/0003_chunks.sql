-- One window of a source's text. `start` and `length` are the machine location
-- and are always there. The text itself is not stored and is read back from the
-- file at those offsets.
--
-- The text is cut twice. A window with no parent is the large one, which is
-- what a result shows, and the windows inside it are what carries a vector.
--
-- `location` is what the source's own numbering calls the place — a chapter, a
-- printed page. It is a projection of whatever the format offered, so it is
-- nullable, and "nothing was written" stays a different answer from "an empty
-- value was written".
--
-- `vault_id` is here because a search constrains on it, and the pair with
-- `source_id` is the foreign key, so a chunk cannot claim a vault its source
-- does not belong to.
CREATE TABLE chunks (
    id        INTEGER PRIMARY KEY,
    source_id INTEGER NOT NULL,
    vault_id  INTEGER NOT NULL,
    start     INTEGER NOT NULL,
    length    INTEGER NOT NULL,
    parent    INTEGER REFERENCES chunks(id) ON DELETE CASCADE,
    location  TEXT,

    FOREIGN KEY (source_id, vault_id) REFERENCES sources (id, vault_id) ON DELETE CASCADE
);

-- The chunks of one source are replaced wholesale, and this is also the key a
-- cascade finds them by.
CREATE INDEX chunks_by_source ON chunks (source_id, vault_id);

-- A cascade finds the windows inside a large one by this key, and reads the
-- whole table without it.
CREATE INDEX chunks_by_parent ON chunks (parent);

-- Which chunks of a vault still owe work, asked from an id onwards so that the
-- answer resumes.
CREATE INDEX chunks_by_vault ON chunks (vault_id, id);

-- The representation the candidates are reranked by.
--
-- `model`, `dims` and `kind` are stored because a blob does not say what it
-- holds: 1024 int8 dimensions and 256 float32 dimensions are the same 1024
-- bytes, and both sides of a comparison name their type.
CREATE TABLE vectors (
    chunk_id INTEGER PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
    model    TEXT NOT NULL,
    dims     INTEGER NOT NULL,
    kind     TEXT NOT NULL,
    v        BLOB NOT NULL
);

-- The coarse pass, one bit per dimension, over everything.
--
-- `chunk_id integer primary key` makes the chunk's own number the rowid of this
-- table. It is the only key a row here can be deleted by without reading the
-- whole index.
--
-- `vault_id` is a metadata column, and a search constrains it inside the query.
-- A nearest-neighbour question is answered over the whole table.
--
-- Nothing cascades into a virtual table, so a chunk's row here is deleted by
-- the code that deletes the chunk.
CREATE VIRTUAL TABLE chunks_vec USING vec0 (
    chunk_id  integer primary key,
    vault_id  integer,
    embedding bit[1024]
);
