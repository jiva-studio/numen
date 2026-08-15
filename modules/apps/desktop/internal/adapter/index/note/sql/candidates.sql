-- Everything a name could mean in this vault, cheapest question first: a path
-- from the root, or a file with that name anywhere.
SELECT path
FROM notes
WHERE vault_id = ?
  AND (path = ? OR path = ? OR basename = ? OR basename = ?)
ORDER BY path;
