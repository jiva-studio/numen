-- The small chunks inside the large chunk that encloses each given chunk of one
-- vault. A given chunk is a small chunk or a large one; a large chunk with no
-- small chunks has no row.
--
-- The ids arrive as one JSON array, which keeps the statement one shape
-- whatever their count is.
SELECT c.id, s.id
FROM json_each(?1) j
JOIN chunks c ON c.id = j.value AND c.vault_id = ?2
JOIN chunks s ON s.vault_id = ?2 AND s.parent_id = COALESCE(c.parent_id, c.id)
ORDER BY j.key, s.id;
