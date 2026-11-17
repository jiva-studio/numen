-- The changes of a vault after one number and up to another, of the events of a JSON array, in the
-- order they were stored, with the path of the source now. A removal, whose number another source may
-- have now, has the path the source had when it was removed.
SELECT c.source_id, CASE WHEN c.event = ? THEN c.path ELSE coalesce(s.path, c.path) END, c.event, c.seq
FROM file_changes c
JOIN vaults v ON v.id = c.vault_id
LEFT JOIN sources s ON s.id = c.source_id
WHERE v.identifier = ? AND c.seq > ? AND c.seq <= ?
  AND c.event IN (SELECT value FROM json_each(?))
ORDER BY c.seq
LIMIT ?;
