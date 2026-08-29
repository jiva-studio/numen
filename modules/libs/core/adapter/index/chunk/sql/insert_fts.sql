-- The chunk's own number is the rowid, which is the only key this table has.
-- Replacing a row by it leaves nothing of the old one in the index.
INSERT OR REPLACE INTO chunks_fts (rowid, text) VALUES (?, ?);
