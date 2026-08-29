-- A chunk whose text is what it was, moved to where that text now is and put
-- inside the large chunk it now sits in.
UPDATE chunks SET start = ?, length = ?, parent = ?, location = ? WHERE id = ?;
