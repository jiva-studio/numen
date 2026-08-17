INSERT INTO vectors (chunk_id, model, dims, kind, v)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (chunk_id) DO UPDATE SET
    model = excluded.model,
    dims  = excluded.dims,
    kind  = excluded.kind,
    v     = excluded.v;
