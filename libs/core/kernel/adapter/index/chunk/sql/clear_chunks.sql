-- Derived rows are replaced wholesale. The vectors go with them; the rows in
-- the vector index are cleared first, because nothing cascades into it.
DELETE FROM chunks WHERE source_id = ?;
