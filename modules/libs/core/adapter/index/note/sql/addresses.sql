-- Every address the notes of this vault point at, each once.
--
-- It is what a sweep asks: what was fetched for an address no note names any
-- more is nobody's, and the folder it stands in is the application's to keep
-- tidy.
SELECT DISTINCT n.address
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND n.address IS NOT NULL AND n.address <> '';
