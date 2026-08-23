-- What one source's text came from: the producer that made it, the hash the
-- files of that reading are kept under, and the file as the index last saw it.
--
-- The size and the time are what say whether the reading is of the bytes on
-- disk now. A file rewritten since is a file whose reading was of other bytes,
-- and coordinates from it fall where those words no longer are.
SELECT COALESCE(text_from, ''), COALESCE(hash, ''), size, modified_at
FROM sources
WHERE vault_id = ? AND path = ?;
