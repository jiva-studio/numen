-- Links written by a name that more than one note answers to.
--
-- Candidates only. Whether such a link is actually ambiguous depends on where
-- it was written — an exact path or a note in the same folder settles it — and
-- that is decided by the same resolution every other link goes through, not
-- here.
SELECT s.path, l.scheme, l.target, l.role,
       COALESCE(l.type, ''), COALESCE(l.why, ''), COALESCE(l.label, '')
FROM links l
JOIN sources s ON s.id = l.note_id
WHERE s.vault_id = ?
  AND l.scheme = 'name'
  AND l.folded_name IN (
    SELECT folded_name FROM notes
    WHERE vault_id = ? GROUP BY folded_name HAVING count(*) > 1
  )
ORDER BY s.path, l.position;
