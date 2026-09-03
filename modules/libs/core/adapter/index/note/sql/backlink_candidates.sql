-- Everything that could point at one note: its identifier, or a name whose last
-- segment is this note's filename.
--
-- Each branch of the union is one index lookup, narrowed afterwards to the vault
-- by the source the link belongs to. A note carrying no identifier is asked for
-- by the empty string, and what that turns up leaves the same way every other
-- candidate does.
--
-- "Could" is the word. Whether a name means this note depends on where the link
-- was written and what else answers to it, so every candidate goes through the
-- same resolution the forward direction uses.
SELECT s.path, l.scheme, l.value, l.role,
       COALESCE(l.type, ''), COALESCE(l.note, ''), COALESCE(l.label, ''), l.position
FROM links l
JOIN sources s ON s.id = l.note_id
WHERE l.scheme = 'note' AND l.value = ? AND s.vault_id = ?
UNION
SELECT s.path, l.scheme, l.value, l.role,
       COALESCE(l.type, ''), COALESCE(l.note, ''), COALESCE(l.label, ''), l.position
FROM links l
JOIN sources s ON s.id = l.note_id
WHERE l.scheme = 'name' AND l.value_base = ? AND s.vault_id = ?
ORDER BY 1, 8;
