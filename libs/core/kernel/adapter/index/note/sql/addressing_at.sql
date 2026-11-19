-- The identifier and folded name of the notes at these paths.
--
-- The paths arrive as a JSON array, so any number of them is one question. A path the index holds
-- no note at is absent from the answer.
SELECT s.path, COALESCE(n.identifier, ''), n.folded_name
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path IN (SELECT value FROM json_each(?));
