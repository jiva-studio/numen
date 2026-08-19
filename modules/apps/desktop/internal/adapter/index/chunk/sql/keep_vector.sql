-- What a model made, kept by the text it read and the recipe it read it under.
--
-- It outlives the chunk that asked for it: chunks are renumbered by every cut,
-- and a vector is bought.
INSERT INTO vectors (fingerprint, recipe, v)
VALUES (?, ?, ?)
ON CONFLICT (fingerprint, recipe) DO UPDATE SET v = excluded.v;
