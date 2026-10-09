-- The chunks of one vault whose text matches the words typed, restricted to
-- chosen source kinds, best first.
SELECT c.id, s.path, s.kind, COALESCE(s.producer, ''), COALESCE(s.hash, ''),
       COALESCE(p.start, c.start),
       COALESCE(p.length, c.length),
       COALESCE(p.location, c.location, ''),
       c.start - COALESCE(p.start, c.start)
FROM chunks_fts
JOIN chunks c ON c.id = chunks_fts.rowid
JOIN sources s ON s.id = c.source_id
LEFT JOIN chunks p ON p.id = c.parent_id
WHERE chunks_fts MATCH ?1
  AND c.vault_id = ?2
  AND s.kind IN (SELECT value FROM json_each(?3))
ORDER BY bm25(chunks_fts)
LIMIT ?4;
