-- What extracting the text produced is recorded on the source: `recipe` names
-- what did it, `hash` addresses the content it read, `producer` names what made
-- the text when the file is not its own.
--
-- All three are set by every write, so a source recorded from a walk alone —
-- which knows none of them — clears what an earlier reading believed. A file
-- that changed is a file whose reading was of other bytes.
INSERT INTO sources (vault_id, path, kind, size, modified_at, hash, recipe, producer)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    size        = excluded.size,
    modified_at = excluded.modified_at,
    hash        = excluded.hash,
    recipe      = excluded.recipe,
    producer    = excluded.producer
RETURNING id;
