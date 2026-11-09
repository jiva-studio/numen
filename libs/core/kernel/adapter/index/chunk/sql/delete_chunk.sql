-- One chunk. The chunks inside it and the vector made from it go with it: the
-- schema says they are the chunk's. The rows in the two virtual tables do not,
-- and are deleted beside this one.
DELETE FROM chunks WHERE id = ?;
