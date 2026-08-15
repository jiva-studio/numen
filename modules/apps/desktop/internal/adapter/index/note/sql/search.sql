SELECT path,
       title,
       snippet(notes_fts, 1, '[', ']', '…', 12)
FROM notes_fts
WHERE notes_fts MATCH ?
  AND vault_id = ?
ORDER BY bm25(notes_fts)
LIMIT ?;
