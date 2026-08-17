-- What extracting the text produced is recorded on the source: `recipe` names
-- what did it, `hash` addresses the content it read.
INSERT INTO sources (vault_id, path, kind, size, modified_at, hash, recipe)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    size        = excluded.size,
    modified_at = excluded.modified_at,
    hash        = excluded.hash,
    recipe      = excluded.recipe
RETURNING id;
