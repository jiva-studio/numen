-- The sources of the vault, and the notes, headings, links, problems and chunks
-- filed under them, go with it: the schema says they are the vault's. The rows
-- in the five virtual tables do not, and are deleted before this runs.
DELETE FROM vaults WHERE id = ?;
