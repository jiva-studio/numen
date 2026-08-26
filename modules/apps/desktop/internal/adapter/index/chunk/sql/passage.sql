-- Where to read one chunk's text from. The text is not stored.
--
-- `parent` is the large chunk this one sits inside, and is zero for a large
-- chunk, which sits inside nothing.
SELECT s.path, COALESCE(s.text_from, ''), COALESCE(s.hash, ''), c.start, c.length, COALESCE(c.location, ''), COALESCE(c.parent, 0)
FROM chunks c
JOIN sources s ON s.id = c.source_id
WHERE c.id = ? AND c.vault_id = ?;
