-- Sources of one kind whose text was not extracted by the recipe in use.
SELECT path
FROM sources
WHERE vault_id = ? AND kind = ? AND (recipe IS NULL OR recipe <> ?)
ORDER BY path
LIMIT ?;
