-- The stored frontmatter of the note at one path, asked once per path.
SELECT s.path, n.frontmatter
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
