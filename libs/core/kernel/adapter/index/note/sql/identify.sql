-- The note's row number, from where the vault files it. A path that names
-- something which is not a note answers nothing.
SELECT n.source_id
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
