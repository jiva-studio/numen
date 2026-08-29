-- The chunks and the vectors go with it: the schema says they are the source's.
-- The rows in the vector index do not, and are deleted before this runs.
DELETE FROM sources WHERE id = ?;
