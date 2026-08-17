-- The vector index is addressed by the chunk's own number, which is its rowid.
-- It is the only key a virtual table has.
DELETE FROM chunks_vec WHERE chunk_id IN (SELECT id FROM chunks WHERE source_id = ?);
