-- One link as it was written. Where it points is a question asked of this table
-- joined with notes: adding a file can resolve a link that was dangling, and
-- deleting one can break a link that worked.
--
-- `note` is what the person wrote about why the link exists.
CREATE TABLE links (
    note_id    INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    scheme     TEXT NOT NULL,
    value      TEXT NOT NULL,

    -- The last segment of the address, without an extension: what
    -- [[notes/Entropy]] and [[Entropy]] have in common, and what the backwards
    -- question is answered through.
    value_base TEXT NOT NULL COLLATE NOCASE,

    role       TEXT NOT NULL,
    type       TEXT,
    note       TEXT,
    label      TEXT,

    PRIMARY KEY (note_id, position)
);

-- A name and an identifier each find few links, which are then narrowed to a
-- vault by the notes they belong to.
CREATE INDEX links_by_target ON links (scheme, value);
CREATE INDEX links_by_name ON links (value_base);

-- What could not be acted on and is worth showing: a link with no role, a
-- target nothing understands. A frontmatter block that could not be read is not
-- here — it belongs to the note it broke, and is kept on the note itself. Both
-- are read together, so a "vault problems" view is a query rather than a
-- rescan.
CREATE TABLE problems (
    note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    detail  TEXT NOT NULL
);

CREATE INDEX problems_by_note ON problems (note_id);
