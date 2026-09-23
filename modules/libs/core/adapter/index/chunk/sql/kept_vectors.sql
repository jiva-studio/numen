-- The vectors already made for these texts under this recipe.
--
-- The hashes arrive as one JSON array of hex, which keeps the statement one
-- shape whatever the count is.
SELECT hex(e.hash), e.embedding
FROM json_each(?) j
JOIN vectors e ON e.hash = unhex(j.value) AND e.recipe = ?;
