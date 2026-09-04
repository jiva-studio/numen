SELECT scheme, value, role, COALESCE(type, ''), COALESCE(why, ''), COALESCE(label, '')
FROM links
WHERE note_id = ?
ORDER BY position;
