-- What one note is called: the name a link written by name finds it by, and the
-- title it is shown under.
UPDATE notes
SET basename = ?, title = ?
WHERE source_id = (SELECT id FROM sources WHERE vault_id = ? AND path = ?);
