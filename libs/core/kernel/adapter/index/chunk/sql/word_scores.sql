-- The negated bm25 of each chunk of the JSON array ?2 whose text matches the
-- FTS5 expression ?1.
SELECT chunks_fts.rowid, -bm25(chunks_fts)
FROM chunks_fts
WHERE chunks_fts MATCH ?1
  AND chunks_fts.rowid IN (SELECT value FROM json_each(?2));
