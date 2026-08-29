-- Small chunks of a vault with no vector under the recipe in use, from one id
-- onwards, so the answer resumes at that id.
--
-- A large chunk carries no vector, and is left out by asking for the ones that
-- sit inside something. A vector is found by the text the chunk holds, so a
-- chunk whose text was embedded under another name is not asked for again.
SELECT c.id, s.path, COALESCE(s.text_from, ''), COALESCE(s.hash, ''), c.start, c.length, COALESCE(c.location, ''), COALESCE(c.parent, 0), c.hash
FROM chunks c
JOIN sources s ON s.id = c.source_id
LEFT JOIN vectors v ON v.fingerprint = unhex(c.hash) AND v.recipe = ?
WHERE c.vault_id = ? AND c.id > ? AND c.parent IS NOT NULL AND v.fingerprint IS NULL
ORDER BY c.id
LIMIT ?;
