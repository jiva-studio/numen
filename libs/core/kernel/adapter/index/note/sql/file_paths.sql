-- The sources of one vault that are not notes: the files a link resolves to by
-- path or by name.
SELECT s.path FROM sources s
WHERE s.vault_id = ?
  AND NOT EXISTS (SELECT 1 FROM notes n WHERE n.source_id = s.id)
ORDER BY s.path;
