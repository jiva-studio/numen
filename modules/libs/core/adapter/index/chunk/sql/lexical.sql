-- A search asked by words: the chunks of one vault whose text matches the words
-- typed, best first.
--
-- The row that comes back is the large chunk enclosing the hit, which is what
-- a result shows, and where the hit itself stands inside it. A chunk with
-- nothing enclosing it is its own, and stands at its own beginning.
--
-- The kinds are a JSON array, and an empty one is every kind: a question that
-- says nothing about what sort of file it wants asks about all of them.
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
  AND (json_array_length(?3) = 0 OR s.kind IN (SELECT value FROM json_each(?3)))
ORDER BY bm25(chunks_fts)
LIMIT ?4;
