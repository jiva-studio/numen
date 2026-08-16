-- Links written by a name that more than one note answers to.
--
-- Candidates only. Whether such a link is actually ambiguous depends on where
-- it was written — an exact path or a note in the same folder settles it — and
-- that is decided by the same resolution every other link goes through, not
-- here.
SELECT n.path, l.scheme, l.value, l.role,
       COALESCE(l.type, ''), COALESCE(l.note, ''), COALESCE(l.label, '')
FROM links l
JOIN notes n ON n.id = l.note_id
WHERE n.vault_id = ?
  AND l.scheme = 'name'
  AND l.value_base IN (
    SELECT basename FROM notes WHERE vault_id = ? GROUP BY basename HAVING count(*) > 1
  )
ORDER BY n.path, l.position;
