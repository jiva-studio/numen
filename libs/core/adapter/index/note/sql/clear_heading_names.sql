-- The index of headings is addressed by the heading's own number, which is its
-- rowid. It runs before the headings go, which is where the numbers are read
-- from.
DELETE FROM headings_fts WHERE rowid IN (SELECT id FROM headings WHERE note_id = ?);
