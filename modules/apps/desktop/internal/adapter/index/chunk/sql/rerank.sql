-- The full-precision vectors of the chunks the coarse pass answered with.
--
-- The ids arrive as one JSON array, which keeps the statement one shape
-- whatever the candidate count is. `chunk_id` is this table's rowid, so each
-- one is a lookup.
--
-- A blob does not say what it holds, so the row says: the width asked for is
-- the width the query has, and a vector of any other model or kind is not
-- comparable with it and is not read.
SELECT v.chunk_id, v.v
FROM json_each(?) j
JOIN vectors v ON v.chunk_id = j.value
WHERE v.dims = ? AND v.kind = 'int8';
