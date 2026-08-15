-- The rowid is the note's own, so the full-text row can be found again without
-- searching for it.
INSERT INTO notes_fts (rowid, title, body, vault_id, path)
VALUES (?, ?, ?, ?, ?);
