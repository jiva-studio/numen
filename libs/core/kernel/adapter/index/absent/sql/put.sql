INSERT INTO absent_files (vault_id, path, size, mtime, hash, provider) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT DO UPDATE SET size = excluded.size, mtime = excluded.mtime, hash = excluded.hash,
    provider = excluded.provider;
