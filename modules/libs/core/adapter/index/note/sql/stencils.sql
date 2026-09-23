-- Every stencil one vault holds, by path.
--
-- A card names the stencil it is cut by, and this is the list that names them
-- all. The vault leads, because a stencil is reachable from the decks of the
-- vault it sits in.
SELECT s.path, n.title
FROM notes n
JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ? AND n.type = ?
ORDER BY s.path;
