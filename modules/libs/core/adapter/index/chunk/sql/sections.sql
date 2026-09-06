-- The sections of one vault whose names match the words typed, best first.
--
-- What a section answers with is the chunk it opens, so a hit on a name is a
-- passage standing where the section begins. The shape is the same as a search
-- asked by words, so both are read back the same way and both fuse into one
-- order.
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
  AND (json_array_length(?3) = 0 OR s.kind IN (SELECT value FROM json_each(?3)))
ORDER BY bm25(sections_fts)
LIMIT ?4;
