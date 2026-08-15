SELECT path, size, mtime
FROM files
WHERE vault_id = ?;
