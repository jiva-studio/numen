-- The note a vault opens on: the first one it holds.
SELECT path, title, COALESCE(identifier, '')
FROM notes
WHERE vault_id = ?
ORDER BY id
LIMIT 1;
