SELECT p.path, p.detail
FROM problems p
WHERE p.vault_id = ?
UNION ALL
SELECT n.path, 'frontmatter: ' || n.frontmatter_err
FROM notes n
WHERE n.vault_id = ? AND n.frontmatter_err IS NOT NULL
ORDER BY 1;
