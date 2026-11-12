-- The index of titles is addressed by the note's own number, which is its rowid.
-- It runs before the vault goes, which is where the numbers are read from.
DELETE FROM titles_fts WHERE rowid IN (SELECT source_id FROM notes WHERE vault_id = ?);
