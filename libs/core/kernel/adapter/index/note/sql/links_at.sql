-- The links written in the notes at these paths, in the order they stand in each note.
--
-- The paths arrive as a JSON array, so any number of notes is one question. A path the index holds
-- no note at is absent from the answer.
SELECT s.path, l.scheme, l.target, l.role,
       COALESCE(l.type, ''), COALESCE(l.why, ''), COALESCE(l.label, ''), COALESCE(l.marker, '')
FROM sources s
JOIN notes n ON n.source_id = s.id
JOIN links l ON l.note_id = n.source_id
WHERE s.vault_id = ? AND s.path IN (SELECT value FROM json_each(?))
ORDER BY s.path, l.position;
