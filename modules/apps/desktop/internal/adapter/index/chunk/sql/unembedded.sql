-- Small windows of a vault with no vector from the model in use, from one id
-- onwards, so the answer resumes at that id.
--
-- A large window carries no vector, and is left out by asking for the ones that
-- sit inside something.
SELECT c.id, s.path, c.start, c.length, COALESCE(c.location, ''), COALESCE(c.parent, 0)
FROM chunks c
JOIN sources s ON s.id = c.source_id
LEFT JOIN chunk_vectors v ON v.chunk_id = c.id AND v.model = ?
WHERE c.vault_id = ? AND c.id > ? AND c.parent IS NOT NULL AND v.chunk_id IS NULL
ORDER BY c.id
LIMIT ?;
