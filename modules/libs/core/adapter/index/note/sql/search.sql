-- The notes whose text matches the words typed, best first.
--
-- The words are indexed over chunks, so a note matching in several of its
-- chunks is grouped back to the one note and ranked by its best chunk.
--
-- `rank` is the full-text table's own score for the row, which is what an
-- aggregate can be taken of.
SELECT s.path, n.title, MIN(chunks_fts.rank) AS score
FROM chunks_fts
JOIN chunks c ON c.id = chunks_fts.rowid
JOIN notes n ON n.source_id = c.source_id
JOIN sources s ON s.id = n.source_id
WHERE chunks_fts MATCH ?
  AND c.vault_id = ?
GROUP BY n.source_id
ORDER BY score
LIMIT ?;
