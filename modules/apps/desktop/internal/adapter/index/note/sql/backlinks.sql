-- Everything pointing at one note, by whichever address form was written: its
-- identifier, or a name that resolves to it.
SELECT l.from_path, l.role, COALESCE(l.type, '')
FROM links l
WHERE l.vault_id = ?
  AND (
        (l.scheme = 'note' AND l.value = ?)
     OR (l.scheme = 'name' AND ? <> '' AND l.value = ?)
  )
ORDER BY l.from_path;
