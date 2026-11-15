-- The paths of the files of the changes of a vault after one number and up to another, each once.
SELECT DISTINCT CASE WHEN c.event = ? THEN c.path ELSE coalesce(s.path, c.path) END
FROM file_changes c
JOIN vaults v ON v.id = c.vault_id
LEFT JOIN sources s ON s.id = c.source_id
WHERE v.identifier = ? AND c.seq > ? AND c.seq <= ?;
