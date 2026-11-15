-- The vectors under the recipe ?3 of the small chunks of the sources of one
-- vault at the paths ?2, a JSON array, after the chunk ?4, in order of chunk id.
SELECT c.id, v.embedding
FROM sources s
JOIN chunks c ON c.source_id = s.id AND c.vault_id = s.vault_id
JOIN vectors v ON v.hash = unhex(c.hash) AND v.recipe = ?3
WHERE s.vault_id = ?1
  AND s.path IN (SELECT value FROM json_each(?2))
  AND c.parent_id IS NOT NULL
  AND c.id > ?4
ORDER BY c.id
LIMIT ?5;
