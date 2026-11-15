-- The sources of one vault under any of the path prefixes and of the kinds,
-- with the title of a note, the size, the time of change in nanoseconds and the
-- count of the small chunks that hold a vector under the recipe, in order of
-- path.
--
-- The prefixes arrive as two JSON arrays, composed (NFC) ?2 and decomposed
-- (NFD) ?3; an empty array limits none, and so does an empty kind array ?4.
-- A negative limit ?6 returns every source. When ?7 is 1, only the sources at
-- the paths of the JSON array ?8 are returned.
SELECT s.path, s.kind, COALESCE(n.title, ''), s.size, s.modified_at,
       (SELECT count(*)
        FROM chunks c
        JOIN vectors v ON v.hash = unhex(c.hash) AND v.recipe = ?5
        WHERE c.source_id = s.id AND c.parent_id IS NOT NULL)
FROM sources s
LEFT JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ?1
  AND (json_array_length(?2) = 0
       OR EXISTS (SELECT 1 FROM json_each(?2) p WHERE substr(s.path, 1, length(p.value)) = p.value)
       OR EXISTS (SELECT 1 FROM json_each(?3) p WHERE substr(s.path, 1, length(p.value)) = p.value))
  AND (json_array_length(?4) = 0 OR s.kind IN (SELECT value FROM json_each(?4)))
  AND (?7 = 0 OR s.path IN (SELECT value FROM json_each(?8)))
ORDER BY s.path
LIMIT ?6;
