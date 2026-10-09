-- Small chunks of a vault with no vector under the recipe in use, ordered so
-- recently modified sources are embedded first.
--
-- A large chunk carries no vector, and is left out by asking for the ones that
-- sit inside something. A vector is found by the text the chunk holds, so a
-- chunk whose text was embedded under another name is not asked for again.
SELECT c.id, s.path, COALESCE(s.producer, ''), COALESCE(s.hash, ''), c.start, c.length, COALESCE(c.location, ''), COALESCE(c.parent_id, 0), c.hash, s.modified_at, s.id
FROM sources s
CROSS JOIN chunks c ON c.source_id = s.id AND c.vault_id = s.vault_id
LEFT JOIN vectors v ON v.hash = unhex(c.hash) AND v.recipe = ?
WHERE s.vault_id = ?
  AND (s.modified_at < ? OR (s.modified_at = ? AND (s.id < ? OR (s.id = ? AND c.id > ?))))
  AND c.parent_id IS NOT NULL
  AND v.hash IS NULL
ORDER BY s.modified_at DESC, s.id DESC, c.id ASC
LIMIT ?;
