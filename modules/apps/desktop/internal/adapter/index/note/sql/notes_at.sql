-- What is needed to show one note, for a path. Asked once per path rather than
-- with a list, so that the statement is one the database can keep.
SELECT path, title, COALESCE(identifier, '')
FROM notes
WHERE vault_id = ? AND path = ?;
