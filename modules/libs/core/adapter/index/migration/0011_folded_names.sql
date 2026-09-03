-- A name is one name whatever case and whatever composition it is written in.
-- `notes.basename` and `links.value_base` hold the key `numen_fold` computes,
-- and the comparison is plain equality on it.
--
-- An index built before this holds the names as they were spelled, under a
-- collation that folds `a` to `A` and nothing else. So `[[энтропия]]` reached
-- no `Энтропия.md`, and a name composed one way reached no file composed the
-- other. The stored names are folded here, which needs no file read.
--
-- The collation comes off both columns, and SQLite changes a collation only by
-- rewriting the table. `notes` is the parent of `headings`, `links` and
-- `problems`, whose rows go with it, so they are set aside first and put back
-- after.

CREATE TABLE headings_kept AS SELECT * FROM headings;
CREATE TABLE links_kept AS SELECT * FROM links;
CREATE TABLE problems_kept AS SELECT * FROM problems;

CREATE TABLE notes_next (
    source_id         INTEGER PRIMARY KEY,
    vault_id          INTEGER NOT NULL,
    basename          TEXT NOT NULL,
    title             TEXT NOT NULL,
    identifier        TEXT,
    frontmatter       TEXT,
    frontmatter_error TEXT,
    type              TEXT NOT NULL DEFAULT 'note',

    FOREIGN KEY (source_id, vault_id) REFERENCES sources (id, vault_id) ON DELETE CASCADE
);

INSERT INTO notes_next (source_id, vault_id, basename, title, identifier,
                        frontmatter, frontmatter_error, type)
SELECT source_id, vault_id, numen_fold(basename), title, identifier,
       frontmatter, frontmatter_error, type
FROM notes;

DROP TABLE notes;
ALTER TABLE notes_next RENAME TO notes;

CREATE INDEX notes_by_basename ON notes (vault_id, basename);
CREATE INDEX notes_by_identifier ON notes (identifier);
CREATE INDEX notes_by_type ON notes (vault_id, type);

DROP TABLE links;

CREATE TABLE links (
    note_id    INTEGER NOT NULL REFERENCES notes(source_id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    scheme     TEXT NOT NULL,
    value      TEXT NOT NULL,
    value_base TEXT NOT NULL,
    role       TEXT NOT NULL,
    type       TEXT,
    note       TEXT,
    label      TEXT,

    PRIMARY KEY (note_id, position)
);

CREATE INDEX links_by_target ON links (scheme, value);
CREATE INDEX links_by_name ON links (value_base);

INSERT INTO links (note_id, position, scheme, value, value_base, role, type, note, label)
SELECT note_id, position, scheme, value, numen_fold(value_base), role, type, note, label
FROM links_kept;

INSERT INTO headings (id, note_id, line, level, text)
SELECT id, note_id, line, level, text FROM headings_kept;

INSERT INTO problems (note_id, detail)
SELECT note_id, detail FROM problems_kept;

DROP TABLE headings_kept;
DROP TABLE links_kept;
DROP TABLE problems_kept;
