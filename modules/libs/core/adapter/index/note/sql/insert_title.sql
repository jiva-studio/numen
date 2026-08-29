-- The note's own number is the rowid, which is the only key this table has.
-- Replacing a row by it leaves nothing of the old one in the index.
INSERT OR REPLACE INTO titles_fts (rowid, text) VALUES (?, ?);
