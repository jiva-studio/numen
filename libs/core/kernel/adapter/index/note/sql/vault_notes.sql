-- Every indexed note of a vault in path order, read through the covering fingerprint index on sources.
SELECT s.path, n.folded_name, n.title, n.type, COALESCE(n.identifier, ''), n.frontmatter
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.kind = 'note'
ORDER BY s.path;
