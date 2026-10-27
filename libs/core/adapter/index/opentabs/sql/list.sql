SELECT id, kind, path, title, state, is_front
FROM open_tabs
WHERE vault = ?
ORDER BY position;
