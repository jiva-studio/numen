-- Returns the source's identity, which the note, its headings, its links, its
-- problems and its chunks are stored against and its full-text row is keyed by.
--
-- `recipe` is the chunking method of the note, and is set by every write. `content_hash` and
-- `producer` are set by every write too, and a note has neither.
-- `file_id` and `content_hash_time` are kept by a write of the row's own hash,
-- size and modification time, and cleared by any other.
INSERT INTO sources (vault_id, path, kind, size, modified_at, content_hash, recipe, producer)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    size        = excluded.size,
    modified_at = excluded.modified_at,
    content_hash = excluded.content_hash,
    recipe      = excluded.recipe,
    producer    = excluded.producer,
    file_id     = CASE WHEN excluded.content_hash = sources.content_hash AND excluded.size = sources.size
                        AND excluded.modified_at = sources.modified_at THEN sources.file_id END,
    content_hash_time = CASE WHEN excluded.content_hash = sources.content_hash AND excluded.size = sources.size
                        AND excluded.modified_at = sources.modified_at THEN sources.content_hash_time END
RETURNING id, (EXISTS (SELECT 1 FROM notes WHERE source_id = sources.id));
