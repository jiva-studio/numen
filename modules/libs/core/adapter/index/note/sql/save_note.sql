INSERT INTO notes (source_id, vault_id, folded_name, title, type, identifier,
                   frontmatter, frontmatter_error)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (source_id) DO UPDATE SET
    folded_name       = excluded.folded_name,
    title             = excluded.title,
    type              = excluded.type,
    identifier        = excluded.identifier,
    frontmatter       = excluded.frontmatter,
    frontmatter_error = excluded.frontmatter_error;
