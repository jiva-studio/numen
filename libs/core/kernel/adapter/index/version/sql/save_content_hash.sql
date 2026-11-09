-- The hash of a file's bytes, recorded with the stat it was taken at. Updated
-- only when the row's size and modification time match the stat the file was
-- hashed at, or the row already has the hash: a file whose content changed since the scan
-- keeps the row's stat, so the indexer still sees the change.
UPDATE sources
SET content_hash = ?, size = ?, modified_at = ?, file_id = ?, content_hash_time = ?
WHERE vault_id = (SELECT id FROM vaults WHERE identifier = ?)
  AND path = ?
  AND ((size = ? AND modified_at = ?) OR content_hash = ?);
