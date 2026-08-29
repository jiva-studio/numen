-- The row number and the path of every source the vault holds at a path and
-- beneath it.
--
-- The path is one lookup, and what is under it is the range from the folder's
-- slash to the byte after it.
SELECT id, path FROM sources
WHERE vault_id = ?
  AND (path = ? OR (path >= ? AND path < ?));
