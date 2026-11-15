-- The leaf chunks of one vault whose text matches the FTS5 expression ?1, with
-- their negated bm25, best first, in the sources at the paths ?3, a JSON array.
SELECT c.id, -bm25(chunks_fts) AS score
FROM chunks_fts
JOIN chunks c ON c.id = chunks_fts.rowid
JOIN sources s ON s.id = c.source_id
WHERE chunks_fts MATCH ?1
  AND c.vault_id = ?2
  AND s.path IN (SELECT value FROM json_each(?3))
ORDER BY score DESC, c.id
LIMIT ?4;
