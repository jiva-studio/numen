SELECT n.path, p.detail
FROM problems p
JOIN notes n ON n.id = p.note_id
WHERE n.vault_id = ?
UNION ALL
SELECT path, 'frontmatter: ' || frontmatter_error
FROM notes
WHERE vault_id = ? AND frontmatter_error IS NOT NULL
ORDER BY 1;
