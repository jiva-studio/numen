-- Returns the source's identity, which the note, its headings, its links, its
-- problems and its chunks are stored against and its full-text row is keyed by.
INSERT INTO sources (vault_id, path, kind, size, modified_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    size        = excluded.size,
    modified_at = excluded.modified_at
RETURNING id;
