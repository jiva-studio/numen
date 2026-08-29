-- The full-text index is addressed by the chunk's own number, which is its
-- rowid. It runs before the chunks go, which is where the numbers are read from.
DELETE FROM chunks_fts WHERE rowid IN (SELECT id FROM chunks WHERE source_id = ?);
