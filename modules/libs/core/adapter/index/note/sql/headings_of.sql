-- What one note is divided into, in the order the headings stand in it. Asked
-- once per path, so the statement is one the database can keep.
SELECT h.line, h.level, h.text
FROM sources s
JOIN notes n ON n.source_id = s.id
JOIN headings h ON h.note_id = n.source_id
WHERE s.vault_id = ? AND s.path = ?
ORDER BY h.line;
