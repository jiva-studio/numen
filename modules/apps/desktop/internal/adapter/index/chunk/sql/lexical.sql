-- The lexical half: the chunks of one vault whose text matches the words typed,
-- best first.
--
-- The row that comes back is the large window enclosing the hit, which is what
-- a result shows, and where the hit itself stands inside it. A window with
-- nothing enclosing it is its own, and stands at its own beginning.
SELECT c.id, s.path, COALESCE(s.text_path, ''),
       COALESCE(p.start, c.start),
       COALESCE(p.length, c.length),
       COALESCE(p.location, c.location, ''),
       c.start - COALESCE(p.start, c.start)
FROM chunks_fts
JOIN chunks c ON c.id = chunks_fts.rowid
JOIN sources s ON s.id = c.source_id
LEFT JOIN chunks p ON p.id = c.parent
WHERE chunks_fts MATCH ?
  AND c.vault_id = ?
ORDER BY bm25(chunks_fts)
LIMIT ?;
