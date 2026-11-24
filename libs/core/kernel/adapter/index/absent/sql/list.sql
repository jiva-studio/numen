SELECT path, size, mtime, hash, provider FROM absent_files WHERE vault_id = ? ORDER BY path;
