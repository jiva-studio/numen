-- The names a vault holds, indexed for the words in them: a note's own title,
-- and every heading inside a note. This is what a person typing in the palette
-- is matched against.
--
-- Two indexes and not one. A title and a heading are each ranked against their
-- own population, and an answer draws every title before every heading, so no
-- score is ever weighed against the other kind.
--
-- Each keeps a copy of the text it indexed, which the chunk index does not. A
-- name is a few words, and what an answer draws is the name with the run that
-- matched marked in it — which is the index saying where it matched, and it can
-- only say that over text it holds.

-- A heading is addressed by a number of its own, so its full-text row can be
-- deleted by it. The table is rebuilt because a column of this kind cannot be
-- added to one that exists.
ALTER TABLE headings RENAME TO headings_unnumbered;

CREATE TABLE headings (
    id       INTEGER PRIMARY KEY,
    note_id  INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    line     INTEGER NOT NULL,
    level    INTEGER NOT NULL,
    text     TEXT NOT NULL,

    UNIQUE (note_id, line)
);

INSERT INTO headings (note_id, line, level, text)
SELECT note_id, line, level, text FROM headings_unnumbered;

DROP TABLE headings_unnumbered;

-- The rowid of each row is what the name belongs to: the note's own number for
-- a title, the heading's own for a heading. It is the only key such a table
-- has, and it is what a name is replaced and deleted by.
CREATE VIRTUAL TABLE titles_fts USING fts5 (text);

CREATE VIRTUAL TABLE headings_fts USING fts5 (text);

-- Both are filled from what the index already holds, so no file is read again.
INSERT INTO titles_fts (rowid, text) SELECT source_id, title FROM notes;

INSERT INTO headings_fts (rowid, text) SELECT id, text FROM headings;
