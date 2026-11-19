-- The source of a vault at a path, and what tells whether its content changed.
SELECT s.id, s.size, s.modified_at, coalesce(s.content_hash, ''), coalesce(n.type, '')
FROM sources s LEFT JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
