-- Everything a name could mean in this vault: a path from the root, the same
-- with an extension, or a file with that name anywhere.
--
-- Each branch of the union is one index lookup. An OR across different columns
-- leaves SQLite able to use only the leading one.
--
-- The folded name is asked for folded, so [[энтропия]] finds Энтропия.md. A
-- path is compared byte for byte, and one written in another case reaches its
-- note through the third branch, where the priority the caller applies folds
-- the path and settles it.
--
-- Only notes answer to a name; the join says so.
SELECT s.path FROM sources s JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?
UNION
SELECT s.path FROM sources s JOIN notes n ON n.source_id = s.id
WHERE s.vault_id = ? AND s.path = ?
UNION
SELECT s.path FROM notes n JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ? AND n.folded_name = ?
ORDER BY 1;
