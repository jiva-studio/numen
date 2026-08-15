-- A result is a title and a path. The text is on disk.
SELECT n.path, n.title
FROM notes_fts
JOIN notes n ON n.id = notes_fts.rowid
WHERE notes_fts MATCH ?
  AND n.vault_id = ?
ORDER BY bm25(notes_fts)
LIMIT ?;
