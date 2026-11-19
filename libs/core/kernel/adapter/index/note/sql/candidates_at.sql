-- Everything each of a set of names could mean in this vault: a path from the root, the same with
-- an extension, or a file with that folded name anywhere.
--
-- The paths (with and without extension) and folded names arrive as two JSON arrays so that any
-- number of names is answered in one query.
SELECT s.path, n.folded_name
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path IN (SELECT value FROM json_each(?))
UNION
SELECT s.path, n.folded_name
FROM notes n
JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ? AND n.folded_name IN (SELECT value FROM json_each(?))
ORDER BY 1;
