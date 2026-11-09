SELECT scheme, target, role, COALESCE(type, ''), COALESCE(why, ''), COALESCE(label, ''), COALESCE(marker, '')
FROM links
WHERE note_id = ?
ORDER BY position;
