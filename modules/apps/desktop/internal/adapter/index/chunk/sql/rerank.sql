-- The full-precision vectors of the chunks the coarse pass answered with.
--
-- The ids arrive as one JSON array, which keeps the statement one shape
-- whatever the candidate count is. `chunk_id` is this table's rowid, so each
-- one is a lookup.
--
-- A blob does not say what it holds, so the row says: the model, the width and
-- the quantisation the query is in. Two models of one width answer questions
-- about different things, and only one of them was asked.
SELECT v.chunk_id, v.v
FROM json_each(?) j
JOIN chunk_vectors v ON v.chunk_id = j.value
WHERE v.model = ? AND v.dims = ? AND v.kind = 'int8';
