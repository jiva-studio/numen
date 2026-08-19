-- The full-precision vectors of the chunks the coarse pass answered with.
--
-- The ids arrive as one JSON array, which keeps the statement one shape
-- whatever the candidate count is. `chunk_id` is this table's rowid, so each
-- one is a lookup.
SELECT v.chunk_id, v.v
FROM json_each(?) j
JOIN vectors v ON v.chunk_id = j.value;
