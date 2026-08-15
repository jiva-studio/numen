SELECT COALESCE(note_id, ''), basename
FROM notes
WHERE vault_id = ? AND path = ?;
