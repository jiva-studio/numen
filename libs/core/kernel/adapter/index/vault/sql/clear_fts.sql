-- The full-text index over chunks is addressed by the chunk's own number, which
-- is its rowid. It runs before the vault goes, which is where the numbers are
-- read from.
DELETE FROM chunks_fts WHERE rowid IN (SELECT id FROM chunks WHERE vault_id = ?);
