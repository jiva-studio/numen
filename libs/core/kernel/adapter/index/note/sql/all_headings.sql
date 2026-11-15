-- Every heading of the notes of the vault, with the path of its note.
SELECT s.path, h.text
FROM headings h
JOIN notes n ON n.source_id = h.note_id
JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ?
ORDER BY s.path;
