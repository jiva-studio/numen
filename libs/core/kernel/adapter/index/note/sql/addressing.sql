-- The two ways a link can arrive at one note: the identifier written in it, and
-- the folded name it is filed under. The name goes straight back as the key a
-- link's own name is matched against, and the name a person reads is computed
-- from the path.
SELECT n.source_id, COALESCE(n.identifier, ''), n.folded_name
FROM sources s
JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?;
