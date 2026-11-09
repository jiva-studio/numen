-- The index of section names is addressed by the chunk that opens the section,
-- whose number is its rowid. It runs before the vault goes, which is where the
-- numbers are read from.
DELETE FROM sections_fts WHERE rowid IN (SELECT id FROM chunks WHERE vault_id = ?);
