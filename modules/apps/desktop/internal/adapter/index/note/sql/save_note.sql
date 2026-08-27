INSERT INTO notes (source_id, vault_id, basename, title, type, identifier,
                   frontmatter, frontmatter_error)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (source_id) DO UPDATE SET
    basename          = excluded.basename,
    title             = excluded.title,
    type              = excluded.type,
    identifier        = excluded.identifier,
    frontmatter       = excluded.frontmatter,
    frontmatter_error = excluded.frontmatter_error;
