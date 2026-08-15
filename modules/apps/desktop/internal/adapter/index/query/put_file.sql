INSERT INTO files (vault_id, path, size, mtime)
VALUES (?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    size  = excluded.size,
    mtime = excluded.mtime;
