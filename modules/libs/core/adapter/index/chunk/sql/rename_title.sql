-- The name one note is ranked and highlighted by. The note's own number is the
-- rowid, which is the only key this table has.
INSERT OR REPLACE INTO titles_fts (rowid, text)
VALUES ((SELECT id FROM sources WHERE vault_id = ? AND path = ?), ?);
