-- Sources of one kind whose text was not extracted by any recipe now in use.
--
-- There is a recipe for every reader, because what took the text out is part of
-- what produced the offsets, and a vault holds files of more than one format.
-- The recipes arrive as a JSON array so that this stays one statement whatever
-- their number.
SELECT path
FROM sources
WHERE vault_id = ? AND kind = ?
  AND (recipe IS NULL OR recipe NOT IN (SELECT value FROM json_each(?)))
ORDER BY path
LIMIT ?;
