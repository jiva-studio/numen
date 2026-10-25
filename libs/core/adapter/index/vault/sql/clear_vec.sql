-- The vector index carries the vault as a metadata column, and every row of one
-- vault is addressed by it. Nothing cascades into a virtual table.
DELETE FROM chunks_vec WHERE vault_id = ?;
