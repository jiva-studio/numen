-- The sections of one vault whose names match the words typed, best first.
--
-- What a section answers with is the chunk it opens, so a hit on a name is a
-- passage standing where the section begins. The shape is the words half's, so
-- both are read back the same way and both can be fused into one order.
SELECT c.id, s.path, COALESCE(s.text_from, ''), COALESCE(s.hash, ''),
       c.start,
       c.length,
       COALESCE(c.location, ''),
       0
FROM parts_fts
JOIN chunks c ON c.id = parts_fts.rowid
JOIN sources s ON s.id = c.source_id
WHERE parts_fts MATCH ?
  AND c.vault_id = ?
ORDER BY bm25(parts_fts)
LIMIT ?;
