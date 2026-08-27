-- What the note at one path is. Asked once per path rather than with a list, so
-- that the statement is one the database can keep.
SELECT n.type
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
