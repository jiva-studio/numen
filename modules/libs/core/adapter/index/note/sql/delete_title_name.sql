-- The index of names is addressed by the note's own number, which is its rowid.
DELETE FROM titles_fts WHERE rowid = ?;
