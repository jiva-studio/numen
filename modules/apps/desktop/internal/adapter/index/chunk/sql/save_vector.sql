-- The chunk is read out of `chunks`, which is where `insert_vec` reads it from
-- too, so the two halves of a vector are both written or neither is.
--
-- A chunk that went while its vector was being made is written nothing, and the
-- run that was making it carries on.
INSERT INTO vectors (chunk_id, model, dims, kind, v)
SELECT id, ?, ?, ?, ? FROM chunks WHERE id = ?
ON CONFLICT (chunk_id) DO UPDATE SET
    model = excluded.model,
    dims  = excluded.dims,
    kind  = excluded.kind,
    v     = excluded.v;
