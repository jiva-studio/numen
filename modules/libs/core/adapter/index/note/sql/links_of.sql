SELECT scheme, value, role, COALESCE(type, ''), COALESCE(note, ''), COALESCE(label, '')
FROM links
WHERE note_id = ?
ORDER BY position;
