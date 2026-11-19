-- The paths of the files of the changes of a vault after one number and up to another, each once.
SELECT DISTINCT CASE WHEN c.event = ?1 THEN c.path ELSE coalesce(s.path, c.path) END, coalesce(t.key, '')
FROM file_changes c
JOIN vaults v ON v.id = c.vault_id
LEFT JOIN sources s ON s.id = c.source_id
LEFT JOIN json_each(c.note_types) t ON t.value > ?3
WHERE v.identifier = ?2 AND c.seq > ?3 AND c.seq <= ?4;
