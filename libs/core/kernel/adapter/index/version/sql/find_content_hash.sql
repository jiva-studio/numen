-- The hash of a file's bytes stored in sources, with the stat it
-- was taken at. A row the scan wrote has no hash, file id or hash time.
SELECT COALESCE(s.content_hash, ''), s.size, s.modified_at, s.file_id, s.content_hash_time
FROM sources s
JOIN vaults v ON v.id = s.vault_id
WHERE v.identifier = ? AND s.path = ?;
