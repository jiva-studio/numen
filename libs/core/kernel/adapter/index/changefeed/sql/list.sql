-- The changes of a vault after one number and up to another, of the events of a JSON array, in the
-- order they were stored, with the path of the source now. A removal, whose number another source may
-- have now, has the path the source had when it was removed and no hash, size or modification time.
SELECT c.source_id, CASE WHEN c.event = ?1 THEN c.path ELSE coalesce(s.path, c.path) END, c.event, c.seq,
       CASE WHEN c.event = ?1 THEN '' ELSE coalesce(s.content_hash, '') END,
       CASE WHEN c.event = ?1 THEN 0 ELSE coalesce(s.size, 0) END,
       CASE WHEN c.event = ?1 THEN 0 ELSE coalesce(s.modified_at, 0) END,
       (SELECT json_group_array(key) FROM json_each(c.note_types) WHERE value > ?3)
FROM file_changes c
JOIN vaults v ON v.id = c.vault_id
LEFT JOIN sources s ON s.id = c.source_id
WHERE v.identifier = ?2 AND c.seq > ?3 AND c.seq <= ?4
  AND c.event IN (SELECT value FROM json_each(?5))
ORDER BY c.seq
LIMIT ?6;
