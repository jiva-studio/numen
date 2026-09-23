-- What is needed to show one note, for a path. Asked once per path rather than
-- with a list, so that the statement is one the database can keep.
SELECT s.path, n.title, COALESCE(n.identifier, '')
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
