-- The chunks, among those given, that belong to a source whose path starts with
-- the prefix, in the order given.
--
-- The ids arrive as one JSON array, which keeps the statement one shape
-- whatever the candidate count is. The prefix is compared as text.
SELECT c.id
FROM json_each(?1) j
JOIN chunks c ON c.id = j.value
JOIN sources s ON s.id = c.source_id
WHERE substr(s.path, 1, length(?2)) = ?2
ORDER BY j.key;
