-- The full-text index is addressed by the chunk's own number, which is its
-- rowid.
DELETE FROM chunks_fts WHERE rowid = ?;
