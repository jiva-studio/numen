-- Deleted by rowid, which is the only key an FTS5 table has. Matching on
-- vault_id and path instead scans the whole index, once per note saved, which
-- makes a rebuild quadratic in the number of notes.
DELETE FROM notes_fts WHERE rowid = ?;
