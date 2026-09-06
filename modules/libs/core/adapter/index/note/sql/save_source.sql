-- Returns the source's identity, which the note, its headings, its links, its
-- problems and its chunks are stored against and its full-text row is keyed by.
--
-- `hash`, `recipe` and `producer` are what a link note's text was made from:
-- the address it points at, what cut the two texts into one, and what fetched
-- the half nobody typed here. All three are set by every write, so a note that
-- points nowhere clears what an earlier fetch believed.
INSERT INTO sources (vault_id, path, kind, size, modified_at, hash, recipe, producer)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    size        = excluded.size,
    modified_at = excluded.modified_at,
    hash        = excluded.hash,
    recipe      = excluded.recipe,
    producer    = excluded.producer
RETURNING id;
