-- The sections of one vault whose names match the words typed, restricted to
-- chosen source kinds, best first. A path prefix limits the sources before the
-- list is truncated to the limit.
WITH hits AS (
    SELECT sections_fts.rowid, bm25(sections_fts) AS rank
    FROM sections_fts
    JOIN chunks c ON c.id = sections_fts.rowid
    JOIN sources s ON s.id = c.source_id
    WHERE sections_fts MATCH ?1
      AND c.vault_id = ?2
      AND s.kind IN (SELECT value FROM json_each(?3))
      AND (substr(s.path, 1, length(?5)) = ?5 OR substr(s.path, 1, length(?6)) = ?6)
    ORDER BY rank
    LIMIT ?4
)
SELECT c.id, s.path, s.kind, COALESCE(s.producer, ''), COALESCE(s.content_hash, ''),
       c.start,
       c.length,
       COALESCE(c.location, ''),
       0
FROM hits h
JOIN chunks c ON c.id = h.rowid
JOIN sources s ON s.id = c.source_id
ORDER BY h.rank;

