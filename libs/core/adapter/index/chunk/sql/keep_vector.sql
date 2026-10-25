-- What a model made, kept by the text it read and the recipe it read it under.
--
-- It outlives the chunk that asked for it: chunks are renumbered by every cut,
-- and a vector is bought. A vector written back as it is kept leaves the row
-- untouched.
INSERT INTO vectors (hash, recipe, embedding)
VALUES (?, ?, ?)
ON CONFLICT (hash, recipe) DO UPDATE SET embedding = excluded.embedding
WHERE embedding IS NOT excluded.embedding;
