-- Everything a name could mean in this vault: a path from the root, the same
-- with an extension, or a file with that name anywhere.
--
-- Each branch of the union is one index lookup. An OR across different columns
-- leaves SQLite able to use only the leading one.
--
-- The basename column folds case, so [[entropy]] finds Entropy.md. A path is
-- compared exactly: it is a path on a disk.
SELECT path FROM notes WHERE vault_id = ? AND path = ?
UNION
SELECT path FROM notes WHERE vault_id = ? AND path = ?
UNION
SELECT path FROM notes WHERE vault_id = ? AND basename = ?
ORDER BY 1;
