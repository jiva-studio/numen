-- Where to read one chunk's text from. The text is not stored.
--
-- `parent_id` is the large chunk this one sits inside, and is zero for a large
-- chunk, which sits inside nothing.
SELECT s.path, COALESCE(s.producer, ''), COALESCE(s.hash, ''), c.start, c.length, COALESCE(c.location, ''), COALESCE(c.parent_id, 0)
FROM chunks c
JOIN sources s ON s.id = c.source_id
WHERE c.id = ? AND c.vault_id = ?;
