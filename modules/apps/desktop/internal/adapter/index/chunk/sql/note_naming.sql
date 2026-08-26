-- What one note is called, and whether anything inside the file says it: a
-- `title` key, or a level-one heading. A note neither names is called by its
-- filename.
SELECT n.title,
       COALESCE(TRIM(json_extract(n.frontmatter, '$.title')), '') <> ''
       OR EXISTS (
           SELECT 1 FROM headings h WHERE h.note_id = n.source_id AND h.level = 1
       )
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
