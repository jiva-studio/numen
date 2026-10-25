-- Every note of one vault that is of one type, by path.
--
-- Nearly every note in a vault is a note, and what this is asked is which of
-- them are the few that are not. The vault leads, because a type means
-- something inside the vault it was written in.
SELECT s.path
FROM notes n
JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ? AND n.type = ?
ORDER BY s.path;
