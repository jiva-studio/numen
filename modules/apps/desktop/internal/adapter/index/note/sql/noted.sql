-- What parsing each file turned up, as the parser said it.
SELECT n.path, p.detail
FROM problems p
JOIN notes n ON n.id = p.note_id
WHERE n.vault_id = ?
ORDER BY n.path, p.rowid;
