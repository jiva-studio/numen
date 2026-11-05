-- Returns the source's identity, which the note, its headings, its links, its
-- problems and its chunks are stored against and its full-text row is keyed by.
--
-- `recipe` is how the note was cut into chunks, and is set by every write. `hash` and
-- `producer` are set by every write too, and a note has neither.
INSERT INTO sources (vault_id, path, kind, size, modified_at, hash, recipe, producer)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    size        = excluded.size,
    modified_at = excluded.modified_at,
    hash        = excluded.hash,
    recipe      = excluded.recipe,
    producer    = excluded.producer
RETURNING id, (EXISTS (SELECT 1 FROM notes WHERE source_id = sources.id));
