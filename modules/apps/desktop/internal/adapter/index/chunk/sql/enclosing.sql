-- The large window one chunk sits inside, and where to read its text from. A
-- window with nothing enclosing it is its own.
SELECT s.path,
       COALESCE(p.start, c.start),
       COALESCE(p.length, c.length),
       COALESCE(p.location, c.location, '')
FROM chunks c
JOIN sources s ON s.id = c.source_id
LEFT JOIN chunks p ON p.id = c.parent
WHERE c.id = ? AND c.vault_id = ?;
