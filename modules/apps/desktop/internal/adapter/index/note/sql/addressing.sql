-- The two ways a link can arrive at one note: the identifier written in it, and
-- the name it is filed under.
SELECT id, COALESCE(identifier, ''), basename
FROM notes
WHERE vault_id = ? AND path = ?;
