-- The value of one top-level frontmatter key in every note of the vault that
-- has it. json_each over the top level compares the key exactly, so a key
-- with a dot or a quote needs no JSON path quoting. A value that is an array
-- or an object arrives as JSON text; type names which.
SELECT s.path, j.type, j.value
FROM notes n
JOIN sources s ON s.id = n.source_id,
     json_each(n.frontmatter) j
WHERE n.vault_id = ? AND n.frontmatter IS NOT NULL AND json_type(n.frontmatter) = 'object' AND j.key = ?
ORDER BY s.path;
