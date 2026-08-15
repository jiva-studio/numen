-- Returns the note's identity, which its headings, links and problems are
-- stored against and its full-text row is keyed by.
INSERT INTO notes (vault_id, path, basename, title, identifier,
                   frontmatter, frontmatter_error, size, modified_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    basename          = excluded.basename,
    title             = excluded.title,
    identifier        = excluded.identifier,
    frontmatter       = excluded.frontmatter,
    frontmatter_error = excluded.frontmatter_error,
    size              = excluded.size,
    modified_at       = excluded.modified_at
RETURNING id;
