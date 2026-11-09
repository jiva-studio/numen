-- What the index believes about every file the vault holds at a path and
-- beneath it.
--
-- Two branches, each one index lookup: the path itself, and the range from the
-- folder's slash to the byte after it, which is every path under the folder.
SELECT path, kind, size, modified_at FROM sources
WHERE vault_id = ? AND path = ?
UNION ALL
SELECT path, kind, size, modified_at FROM sources
WHERE vault_id = ? AND path >= ? AND path < ?
ORDER BY 1;
