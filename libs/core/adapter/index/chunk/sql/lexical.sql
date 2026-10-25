-- A search asked by words: the chunks of one vault whose text matches the words
-- typed, best first.
--
-- The row that comes back is the large chunk enclosing the hit, which is what
-- a result shows, and where the hit itself stands inside it. A chunk with
-- nothing enclosing it is its own, and stands at its own beginning.
WITH hits AS (
    SELECT chunks_fts.rowid, bm25(chunks_fts) AS rank
    FROM chunks_fts
    JOIN chunks c ON c.id = chunks_fts.rowid
    WHERE chunks_fts MATCH ?1
      AND c.vault_id = ?2
    ORDER BY rank
    LIMIT ?3
)
SELECT c.id, s.path, s.kind, COALESCE(s.producer, ''), COALESCE(s.hash, ''),
       COALESCE(p.start, c.start),
       COALESCE(p.length, c.length),
       COALESCE(p.location, c.location, ''),
       c.start - COALESCE(p.start, c.start)
FROM hits h
JOIN chunks c ON c.id = h.rowid
JOIN sources s ON s.id = c.source_id
LEFT JOIN chunks p ON p.id = c.parent_id
ORDER BY h.rank;

