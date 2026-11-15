-- The leaf chunks of one vault whose text matches the FTS5 expression ?1, with
-- their negated bm25, best first, under any of the path prefixes and of the
-- kinds. The prefixes arrive as JSON arrays, composed (NFC) ?3 and decomposed
-- (NFD) ?4; an empty array limits none, and so does an empty kind array ?5.
SELECT c.id, -bm25(chunks_fts) AS score
FROM chunks_fts
JOIN chunks c ON c.id = chunks_fts.rowid
JOIN sources s ON s.id = c.source_id
WHERE chunks_fts MATCH ?1
  AND c.vault_id = ?2
  AND (json_array_length(?3) = 0
       OR EXISTS (SELECT 1 FROM json_each(?3) p WHERE substr(s.path, 1, length(p.value)) = p.value)
       OR EXISTS (SELECT 1 FROM json_each(?4) p WHERE substr(s.path, 1, length(p.value)) = p.value))
  AND (json_array_length(?5) = 0 OR s.kind IN (SELECT value FROM json_each(?5)))
ORDER BY score DESC, c.id
LIMIT ?6;
