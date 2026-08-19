-- The large window one chunk sits inside, where to read its text from, and
-- where the chunk itself stands inside that window. A window with nothing
-- enclosing it is its own, and stands at its own beginning.
SELECT s.path, COALESCE(s.text_path, ''),
       COALESCE(p.start, c.start),
       COALESCE(p.length, c.length),
       COALESCE(p.location, c.location, ''),
       c.start - COALESCE(p.start, c.start)
FROM chunks c
JOIN sources s ON s.id = c.source_id
LEFT JOIN chunks p ON p.id = c.parent
WHERE c.id = ? AND c.vault_id = ?;
