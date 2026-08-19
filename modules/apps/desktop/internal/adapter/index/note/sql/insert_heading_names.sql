-- Every heading of one note, indexed for the words in it, once they are all
-- stored and carry the numbers this table is keyed by.
INSERT INTO headings_fts (rowid, text) SELECT id, text FROM headings WHERE note_id = ?;
