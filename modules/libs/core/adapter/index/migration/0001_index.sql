-- A vault the index knows about.
--
-- `identifier` is the ULID in the vault's own config file, which travels with
-- the folder. `id` is what this database calls the vault, and is what every
-- other row here carries.
CREATE TABLE vaults (
    id         INTEGER PRIMARY KEY,
    identifier TEXT NOT NULL UNIQUE
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
--
-- `producer` is which producer made the text this source's chunks are places
-- in. A scanned document holds no text a machine can take out of it: what
-- reading it produced is a file of its own, and showing a passage reads that
-- file. Null is the ordinary case, and the only case for a note or a book whose
-- text is its own.
--
-- `hash`, `recipe` and `producer` are one fact and are cleared by one write:
-- the file is not the file that was read.
CREATE TABLE sources (
    id          INTEGER PRIMARY KEY,
    vault_id    INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    path        TEXT NOT NULL,
    kind        TEXT NOT NULL,
    size        INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    hash        TEXT,
    recipe      TEXT,
    producer    TEXT,

    UNIQUE (vault_id, path)
);

-- Every row filed under a vault carries the vault, and names the pair as its
-- foreign key, so a row cannot claim a vault its source does not belong to.
CREATE UNIQUE INDEX sources_by_vault ON sources (id, vault_id);

-- Every scan asks one question of every source of one kind: has this file
-- changed. This answers it without reading the files themselves.
CREATE INDEX sources_by_fingerprint ON sources (vault_id, kind, path, size, modified_at);

-- Which sources of a vault stand on a text a producer made. It is asked once a
-- scan, to find the ones whose file a person deleted by hand, and it is answered
-- in proportion to the documents that were read rather than to the library.
CREATE INDEX sources_by_text ON sources (vault_id, kind) WHERE producer IS NOT NULL;

-- What only a note has: the names a link reaches it by, and the frontmatter
-- they are written in. A note's row number is its source's, so a question that
-- needs only the file joins `sources` on it.
--
-- `identifier` is the ULID written in the file, when there is one. Most notes
-- have none, and a copied file carries a copy of it.
--
-- `type` is what the note is: a note, a deck of cards, the stencil a card is cut
-- by, or the preset a deck is scheduled under. The list is closed, and a note
-- whose file says nothing is a note.
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

-- A name means something inside one vault, so the vault leads. The path a
-- candidate is reported by comes from the source, which is one lookup by row
-- number away.
CREATE INDEX notes_by_folded_name ON notes (vault_id, folded_name);

-- An identifier names one note in the world, and is looked up without a vault.
CREATE INDEX notes_by_identifier ON notes (identifier);

-- Which notes of a vault are of one kind. Nearly every note in a vault is a
-- note, and what this is asked is which of them are the few that are not.
CREATE INDEX notes_by_type ON notes (vault_id, type);

-- A heading inside a note. It is addressed by a number of its own, because its
-- full-text row is keyed by one.
CREATE TABLE headings (
    id       INTEGER PRIMARY KEY,
    note_id  INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    line     INTEGER NOT NULL,
    level    INTEGER NOT NULL,
    text     TEXT NOT NULL,

    UNIQUE (note_id, line)
);

-- One link as it was written. Where it points is a question asked of this table
-- joined with notes: adding a file can resolve a link that was dangling, and
-- deleting one can break a link that worked.
--
-- `note` is what the person wrote about why the link exists.
CREATE TABLE links (
    note_id     INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    scheme      TEXT NOT NULL,
    value       TEXT NOT NULL,

    -- The last segment of the address under the same fold a note's name is
    -- held to, which is what the two are compared on: what [[notes/Entropy]],
    -- [[Entropy]] and [[entropy.MD]] have in common.
    folded_name TEXT NOT NULL,

    role        TEXT NOT NULL,
    type        TEXT,
    note        TEXT,
    label       TEXT,

    PRIMARY KEY (note_id, position)
);

-- A name and an identifier each find few links, which are then narrowed to a
-- vault by the notes they belong to.
CREATE INDEX links_by_target ON links (scheme, value);
CREATE INDEX links_by_name ON links (folded_name);

-- What could not be acted on and is worth showing: a link with no role, a
-- target nothing understands. A frontmatter block that could not be read is not
-- here — it belongs to the note it broke, and is kept on the note itself. Both
-- are read together, so a "vault problems" view is a query rather than a
-- rescan.
CREATE TABLE problems (
    note_id INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    detail  TEXT NOT NULL
);

CREATE INDEX problems_by_note ON problems (note_id);

-- One chunk of a source's text. `start` and `length` are the machine location
-- and are always there. The text itself is not stored and is read back from the
-- file at those offsets.
--
-- The text is cut twice. A chunk with no parent is the large one, which is
-- what a result shows, and the chunks inside it are what carries a vector.
--
-- `hash` is the address the chunk's text gives it, and is what identifies the
-- chunk. Cutting a source again is a comparison against this column: a chunk
-- whose hash is already on a row keeps that row, and its full-text row with it.
-- It is also how the vector made from that text is found.
--
-- `location` is where the chunk sits in the source's own numbering — a chapter,
-- a printed page. It is a projection of whatever the format offered, so it is
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
    hash      TEXT NOT NULL,

    FOREIGN KEY (source_id, vault_id) REFERENCES sources (id, vault_id) ON DELETE CASCADE
);

-- The chunks of one source are replaced wholesale, and this is also the key a
-- cascade finds them by.
CREATE INDEX chunks_by_source ON chunks (source_id, vault_id);

-- A cascade finds the chunks inside a large one by this key, and reads the
-- whole table without it.
CREATE INDEX chunks_by_parent ON chunks (parent);

-- Which chunks of a vault still owe work, asked from an id onwards so that the
-- answer resumes.
CREATE INDEX chunks_by_vault ON chunks (vault_id, id);

-- How far embedding has got is asked of the chunks that carry vectors. A chunk
-- that encloses others carries none, so the total counts only the ones that do.
CREATE INDEX chunks_by_vault_parent ON chunks (vault_id, parent, id);

-- Finding a chunk by the text it holds, which is how a vector is claimed and
-- how a vector nothing holds any more is recognised.
CREATE INDEX chunks_by_hash ON chunks (hash);

-- What a model made, addressed by the text it read and the recipe it read it
-- under.
--
-- Everything else this index holds is a reading of files that are still on
-- disk, and a scan puts it back for the cost of reading them. A vector is
-- bought: minutes of a machine, or money and a network. It is kept where no
-- renumbering of chunks and no rebuilding of the index can reach it.
--
-- The recipe names everything that decides what the vector is: where it was
-- made, which model, how wide, where the text was cut off, how the model's
-- output becomes one vector, and how the numbers are stored. Change any of them
-- and the old rows are simply not found, still here, still there if the setting
-- goes back.
--
-- A rowid table with the key in an index of its own: the vector is a kilobyte,
-- and a key that carries it is a key every probe reads a kilobyte to answer.
CREATE TABLE vectors (
    hash   BLOB NOT NULL,
    recipe TEXT NOT NULL,
    vector BLOB NOT NULL
);

CREATE UNIQUE INDEX vectors_by_hash ON vectors (hash, recipe);

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
-- the code that deletes the chunk. The width is the model's, and a model of
-- another width rebuilds this table from what has been made.
CREATE VIRTUAL TABLE chunks_vec USING vec0 (
    chunk_id  integer primary key,
    vault_id  integer,
    embedding bit[1024]
);

-- The full-text index, over chunks. A lexical hit and a dense hit name the same
-- row, so both can be placed in one list and read back the same way.
--
-- `content=''` keeps no copy of what was indexed: the text is on disk, and a
-- chunk is where to read it from. `contentless_delete` is what makes a row
-- replaceable without that copy, which a source cut again needs.
--
-- The rowid is the chunk's own id. It is the only key this kind of table has.
CREATE VIRTUAL TABLE chunks_fts USING fts5 (
    text,
    content='',
    contentless_delete=1
);

-- The names of the parts a source divides into: a section is something a person
-- finds, and not only a label an answer carries.
--
-- The rowid is the chunk the part opens, so a hit on a name is a passage
-- standing at the start of the section, read back the way every passage is. A
-- chunk that opens a part and a subsection under it carries both names, which is
-- one row holding two lines.
--
-- `content=''` keeps no copy: nothing reads a name back. What an answer shows is
-- the chunk's own location.
CREATE VIRTUAL TABLE parts_fts USING fts5 (
    text,
    content='',
    contentless_delete=1
);

-- The names a vault holds, indexed for the words in them: a note's own title,
-- and every heading inside a note. This is what a person typing in the palette
-- is matched against.
--
-- Two indexes. A title and a heading are each ranked against their own
-- population, and an answer draws every title before every heading, so no score
-- is ever weighed against the other kind.
--
-- Each keeps a copy of the text it indexed. What an answer draws is the name
-- with the run that matched marked in it, and the index can only say where it
-- matched over text it holds.
--
-- The rowid of each row is what the name belongs to: the note's own number for
-- a title, the heading's own for a heading. It is the only key such a table
-- has, and it is what a name is replaced and deleted by.
CREATE VIRTUAL TABLE titles_fts USING fts5 (text);

CREATE VIRTUAL TABLE headings_fts USING fts5 (text);
