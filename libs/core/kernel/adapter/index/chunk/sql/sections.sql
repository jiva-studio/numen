-- The sections of one vault whose names match the words typed, best first.
--
-- What a section answers with is the chunk it opens, so a hit on a name is a
-- passage standing where the section begins. The shape is the same as a search
-- asked by words, so both are read back the same way and both fuse into one
-- order. A path prefix limits the sources before the list is truncated to the limit.
WITH hits AS (
    SELECT sections_fts.rowid, bm25(sections_fts) AS rank
    FROM sections_fts
    JOIN chunks c ON c.id = sections_fts.rowid
    JOIN sources s ON s.id = c.source_id
    WHERE sections_fts MATCH ?1
      AND c.vault_id = ?2
      AND (substr(s.path, 1, length(?4)) = ?4 OR substr(s.path, 1, length(?5)) = ?5)
    ORDER BY rank
    LIMIT ?3
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

