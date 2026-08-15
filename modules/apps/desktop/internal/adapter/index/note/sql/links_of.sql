SELECT scheme, value, role, COALESCE(type, ''), COALESCE(reason, ''), COALESCE(label, '')
FROM links
WHERE note_id = ?
ORDER BY position;
