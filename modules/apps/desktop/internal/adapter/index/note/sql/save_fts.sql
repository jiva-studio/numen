-- Replacing a row by its rowid leaves nothing of the old one in the index.
INSERT OR REPLACE INTO notes_fts (rowid, title, body) VALUES (?, ?, ?);
