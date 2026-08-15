INSERT INTO notes (vault_id, path, title, frontmatter, frontmatter_err, note_id, basename)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault_id, path) DO UPDATE SET
    title           = excluded.title,
    frontmatter     = excluded.frontmatter,
    frontmatter_err = excluded.frontmatter_err,
    note_id         = excluded.note_id,
    basename        = excluded.basename;
