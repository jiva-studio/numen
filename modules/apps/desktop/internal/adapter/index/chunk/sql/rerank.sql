-- The vectors of the chunks the coarse pass answered with.
--
-- The ids arrive as one JSON array, which keeps the statement one shape
-- whatever the candidate count is.
--
-- A vector is found by the text its chunk holds, under the recipe the query is
-- in. Two models of one width answer questions about different things, and only
-- one of them was asked.
SELECT c.id, v.v
FROM json_each(?) j
JOIN chunks c ON c.id = j.value
JOIN vectors v ON v.fingerprint = unhex(c.hash) AND v.recipe = ?;
