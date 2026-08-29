-- The vectors already made for these texts under this recipe.
--
-- The fingerprints arrive as one JSON array of hex, which keeps the statement
-- one shape whatever the count is.
SELECT hex(e.fingerprint), e.v
FROM json_each(?) j
JOIN vectors e ON e.fingerprint = unhex(j.value) AND e.recipe = ?;
