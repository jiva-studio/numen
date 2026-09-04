-- Links that reach nothing.
--
-- A link with no candidate at all is dangling whatever the priority rules would
-- have done with it, so this one question is the whole answer and no resolving
-- follows it.
--
-- A name is looked for as a path and as a filename, one index lookup each. The
-- third shape resolution uses — the name with an extension added — needs no
-- branch here: a note it would find has that name as its folded name, so the
-- second condition already covers it.
--
-- Only names are asked about. An identifier no vault here holds is not
-- dangling: it names one note in the world, and the vault holding it may simply
-- not be open on this machine. Reporting it would tell somebody to mend a link
-- that is fine everywhere they use it.
SELECT s.path, l.scheme, l.value, l.role,
       COALESCE(l.type, ''), COALESCE(l.note, ''), COALESCE(l.label, '')
FROM links l
JOIN sources s ON s.id = l.note_id
WHERE s.vault_id = ?
  AND l.scheme = 'name'
  AND NOT EXISTS (
    SELECT 1 FROM sources t
    JOIN notes tn ON tn.source_id = t.id
    WHERE t.vault_id = s.vault_id AND t.path = l.value
  )
  AND NOT EXISTS (
    SELECT 1 FROM notes tn
    WHERE tn.vault_id = s.vault_id AND tn.folded_name = l.folded_name
  )
ORDER BY s.path, l.position;
