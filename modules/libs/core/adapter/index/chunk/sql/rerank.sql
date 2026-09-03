-- The vectors of the chunks the coarse pass answered with.
--
-- The ids arrive as one JSON array, which keeps the statement one shape
-- whatever the candidate count is.
--
-- A vector is found by the text its chunk holds, under the recipe the query is
-- in. Two models of one width answer questions about different things, and only
-- one of them was asked.
--
-- A kind is a filter over what the coarse pass found, and not over what it
-- looked at: a vault holding little of the kind asked for answers with fewer
-- passages than were asked for.
SELECT c.id, v.vector
FROM json_each(?1) j
JOIN chunks c ON c.id = j.value
JOIN sources s ON s.id = c.source_id
JOIN vectors v ON v.fingerprint = unhex(c.hash) AND v.recipe = ?2
WHERE json_array_length(?3) = 0 OR s.kind IN (SELECT value FROM json_each(?3));
