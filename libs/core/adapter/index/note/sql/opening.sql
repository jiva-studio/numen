-- The note a vault opens on: the first one it holds.
SELECT s.path, n.title, COALESCE(n.identifier, '')
FROM notes n
JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ?
ORDER BY n.source_id
LIMIT 1;
