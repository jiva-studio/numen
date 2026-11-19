-- Everything that could point at any of a set of notes: their identifiers, or names whose last
-- segment matches one of their filenames.
--
-- Each branch of the union is one index lookup over a JSON array, narrowed afterwards to the vault
-- by the source the link belongs to.
SELECT s.path, l.scheme, l.target, l.role,
       COALESCE(l.type, ''), COALESCE(l.why, ''), COALESCE(l.label, ''), COALESCE(l.marker, ''), l.position
FROM links l
JOIN sources s ON s.id = l.note_id
WHERE l.scheme = 'note' AND l.target IN (SELECT value FROM json_each(?)) AND s.vault_id = ?
UNION
SELECT s.path, l.scheme, l.target, l.role,
       COALESCE(l.type, ''), COALESCE(l.why, ''), COALESCE(l.label, ''), COALESCE(l.marker, ''), l.position
FROM links l
JOIN sources s ON s.id = l.note_id
WHERE l.scheme = 'name' AND l.folded_name IN (SELECT value FROM json_each(?)) AND s.vault_id = ?
ORDER BY 1, 9;
