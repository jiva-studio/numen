-- The headings of the notes of the vault whose words hold the phrase, with
-- the path of their note.
SELECT s.path, h.text
FROM headings_fts f
JOIN headings h ON h.id = f.rowid
JOIN notes n ON n.source_id = h.note_id
JOIN sources s ON s.id = n.source_id
WHERE headings_fts MATCH ? AND n.vault_id = ?
ORDER BY s.path;
