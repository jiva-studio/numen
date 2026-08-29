-- What the notes at these paths are.
--
-- The paths arrive as a JSON array, so a folder is one question whatever it
-- holds. A path the index holds no note at is absent from the answer.
SELECT s.path, n.type
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path IN (SELECT value FROM json_each(?));
