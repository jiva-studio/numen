-- Every link written in the notes of a vault in source path and declaration order.
SELECT s.path, l.scheme, l.target, l.role,
       COALESCE(l.type, ''), COALESCE(l.why, ''), COALESCE(l.label, ''), COALESCE(l.marker, ''), l.folded_name
FROM sources s
JOIN links l ON l.note_id = s.id
WHERE s.vault_id = ?
ORDER BY s.path, l.position;
