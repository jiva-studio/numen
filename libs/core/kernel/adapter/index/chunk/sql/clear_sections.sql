-- Every section name of one source, dropped before the source is cut again. A
-- chunk keeps its row across a cut, so a name is not dropped with the chunk.
DELETE FROM sections_fts WHERE rowid IN (SELECT id FROM chunks WHERE source_id = ?);
