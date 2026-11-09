-- A chunk whose text is what it was, moved to where that text now is and put
-- inside the large chunk it now is in.
UPDATE chunks SET start = ?, length = ?, parent_id = ?, location = ? WHERE id = ?;
