SELECT scheme, value, role, COALESCE(type, ''), COALESCE(note, ''), COALESCE(label, '')
FROM links
WHERE vault_id = ? AND from_path = ?
ORDER BY position;
