-- The note of one vault that carries an identifier.
SELECT s.path
FROM notes n
JOIN sources s ON s.id = n.source_id
WHERE n.identifier = ? AND n.vault_id = ?
ORDER BY s.path;
