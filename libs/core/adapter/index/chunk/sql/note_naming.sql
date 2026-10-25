-- What one note is called, and whether the file says it with a `title` key. A
-- note that carries no key is called by its filename.
SELECT n.title,
       COALESCE(TRIM(json_extract(n.frontmatter, '$.title')), '') <> ''
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
