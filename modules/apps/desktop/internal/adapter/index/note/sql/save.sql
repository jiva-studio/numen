INSERT INTO notes (vault_id, path, title, frontmatter, frontmatter_err)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    title           = excluded.title,
    frontmatter     = excluded.frontmatter,
    frontmatter_err = excluded.frontmatter_err;
