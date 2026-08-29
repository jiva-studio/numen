-- The two ways a link can arrive at one note: the identifier written in it, and
-- the name it is filed under.
SELECT n.source_id, COALESCE(n.identifier, ''), n.basename
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
