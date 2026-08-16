-- Everything that could point at one note: its identifier, or a name whose last
-- segment is this note's filename.
--
-- Each branch of the union is one index lookup, narrowed afterwards to the vault
-- by the note the link belongs to. An empty identifier is filtered out by the
-- caller rather than by a term here, which the planner could not fold.
--
-- "Could" is the word. Whether a name means this note depends on where the link
-- was written and what else answers to it, so every candidate goes through the
-- same resolution the forward direction uses.
SELECT n.path, l.scheme, l.value, l.role,
       COALESCE(l.type, ''), COALESCE(l.note, ''), COALESCE(l.label, ''), l.position
FROM links l
JOIN notes n ON n.id = l.note_id
WHERE l.scheme = 'note' AND l.value = ? AND n.vault_id = ?
UNION
SELECT n.path, l.scheme, l.value, l.role,
       COALESCE(l.type, ''), COALESCE(l.note, ''), COALESCE(l.label, ''), l.position
FROM links l
JOIN notes n ON n.id = l.note_id
WHERE l.scheme = 'name' AND l.value_base = ? AND n.vault_id = ?
ORDER BY 1, 8;
