-- The sources of one kind whose text is a file of their own rather than their
-- own bytes, and the name of that file.
SELECT path, text_path
FROM sources
WHERE vault_id = ? AND kind = ? AND text_path IS NOT NULL
ORDER BY path;
