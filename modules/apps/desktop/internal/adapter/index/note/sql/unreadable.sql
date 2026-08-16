-- The notes whose frontmatter is not YAML, and what the parser said about it.
SELECT path, frontmatter_error
FROM notes
WHERE vault_id = ? AND frontmatter_error IS NOT NULL
ORDER BY path;
