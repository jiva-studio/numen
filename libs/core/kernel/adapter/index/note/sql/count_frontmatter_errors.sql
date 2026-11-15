-- The count of the notes of the vault whose frontmatter is not readable.
SELECT count(*) FROM notes WHERE vault_id = ? AND frontmatter_error IS NOT NULL;
