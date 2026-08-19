-- What extracting the text produced is recorded on the source: `recipe` names
-- what did it, `hash` addresses the content it read, `text_path` says where the
-- text is when it is not the file itself.
--
-- All three are set by every write, so a source recorded from a walk alone —
-- which knows none of them — clears what an earlier reading believed. A file
-- that changed is a file whose reading was of other bytes.
INSERT INTO sources (vault_id, path, kind, size, modified_at, hash, recipe, text_path)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    size        = excluded.size,
    modified_at = excluded.modified_at,
    hash        = excluded.hash,
    recipe      = excluded.recipe,
    text_path   = excluded.text_path
RETURNING id;
