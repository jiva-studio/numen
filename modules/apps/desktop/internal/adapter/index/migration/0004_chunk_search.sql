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

DROP TABLE IF EXISTS notes_fts;

-- Nothing rebuilds a contentless index from itself, because the text it held
-- was never stored. Removing every source is what makes the next scan read
-- every file again, and reading a file is what fills this table. The rows in
-- the vector index go by hand, since nothing cascades into a virtual table.
DELETE FROM chunks_vec;
DELETE FROM sources;
