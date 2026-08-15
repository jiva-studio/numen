-- Everything that could point at one note: its identifier, or a name whose last
-- segment is this note's filename.
--
-- A union rather than a chain of ORs, and no guard clause inside either branch:
-- an OR across columns, or a comparison the planner cannot fold, leaves SQLite
-- narrowing to the vault and then reading every link in it — once per note
-- opened. Each branch here is an index lookup. An empty identifier is filtered
-- out by the caller rather than by a term in the query.
--
-- "Could" is the word. Whether a name actually means this note depends on where
-- the link was written and what else answers to it, so every candidate is put
-- through the same resolution the forward direction uses. Matching the written
-- text alone reports links that resolve elsewhere and misses links written as a
-- path.
SELECT from_path, scheme, value, role, COALESCE(type, ''), COALESCE(note, ''), COALESCE(label, ''), position
FROM links
WHERE vault_id = ? AND scheme = 'note' AND value = ?
UNION
SELECT from_path, scheme, value, role, COALESCE(type, ''), COALESCE(note, ''), COALESCE(label, ''), position
FROM links
WHERE vault_id = ? AND scheme = 'name' AND value_base = ?
ORDER BY 1, 8;
