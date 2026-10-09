-- The sections of one vault whose names match the words typed, restricted to
-- chosen source kinds, best first.
SELECT c.id, s.path, s.kind, COALESCE(s.producer, ''), COALESCE(s.hash, ''),
       c.start,
       c.length,
       COALESCE(c.location, ''),
       0
FROM sections_fts
JOIN chunks c ON c.id = sections_fts.rowid
JOIN sources s ON s.id = c.source_id
WHERE sections_fts MATCH ?1
  AND c.vault_id = ?2
  AND s.kind IN (SELECT value FROM json_each(?3))
ORDER BY bm25(sections_fts)
LIMIT ?4;
