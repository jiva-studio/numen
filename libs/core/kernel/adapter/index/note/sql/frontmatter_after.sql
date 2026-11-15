-- The stored frontmatter of the notes of one vault after a path, in path order.
SELECT s.path, n.frontmatter
FROM notes n
JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ? AND s.path > ?
ORDER BY s.path
LIMIT ?;
