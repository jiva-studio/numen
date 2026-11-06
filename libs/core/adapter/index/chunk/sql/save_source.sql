-- What extracting the text produced is recorded on the source: `recipe` names
-- what did it, `content_hash` addresses the content it read, `producer` names what made
-- the text when the file is not its own.
--
-- All three are set by every write, so a source recorded from a walk alone —
-- which knows none of them — clears what an earlier reading believed. A file
-- that changed is a file whose recognition was of other bytes.
--
-- `file_id` and `content_hash_time` belong to the hash a lookup of the file
-- recorded. A write that carries the row's own hash, size and modification time
-- keeps them; any other write clears them.
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
RETURNING id;
