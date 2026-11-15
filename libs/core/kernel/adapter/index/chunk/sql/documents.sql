-- The sources of one vault under any of the path prefixes, of the kinds and
-- at the paths, with the title and the type of a note, the time of the last
-- change, the size and the count of the small chunks that hold a vector under
-- the recipe, in order of path.
--
-- The prefixes arrive as two JSON arrays, composed (NFC) ?2 and decomposed
-- (NFD) ?3; an empty array limits none, and so do an empty kind array ?4 and
-- an empty path array ?7. A negative limit ?6 returns every source.
SELECT s.path, s.kind, COALESCE(n.title, ''), COALESCE(n.type, ''), s.modified_at, s.size,
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
  AND (json_array_length(?7) = 0 OR s.path IN (SELECT value FROM json_each(?7)))
ORDER BY s.path
LIMIT ?6;
