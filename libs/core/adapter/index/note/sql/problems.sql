-- What parsing each file turned up, as the parser said it.
SELECT s.path, p.detail
FROM problems p
JOIN sources s ON s.id = p.note_id
WHERE s.vault_id = ?
ORDER BY s.path, p.rowid;
