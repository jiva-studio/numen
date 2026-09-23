-- The notes whose frontmatter is not YAML, and what the parser said about it.
SELECT s.path, n.frontmatter_error
FROM notes n
JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ? AND n.frontmatter_error IS NOT NULL
ORDER BY s.path;
