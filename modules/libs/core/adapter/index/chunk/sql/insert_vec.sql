-- The vault comes from the chunk, so the two cannot disagree.
--
-- `vec_bit` says the blob is one bit per dimension. Its length alone does not
-- distinguish that from float32.
INSERT INTO chunks_vec (chunk_id, vault_id, embedding)
SELECT id, vault_id, vec_bit(?) FROM chunks WHERE id = ?;
