-- The links of the notes of one vault whose target's last segment folds to a
-- name, by the index links_by_name.
SELECT s.path, l.scheme, l.target, l.role,
       COALESCE(l.type, ''), COALESCE(l.why, ''), COALESCE(l.label, ''), COALESCE(l.marker, '')
FROM links l
JOIN sources s ON s.id = l.note_id
WHERE l.folded_name = ? AND s.vault_id = ?
ORDER BY s.path, l.position;
