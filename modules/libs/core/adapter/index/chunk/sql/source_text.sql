-- What one source's text came from: the producer that made it, the hash the
-- files of that recognition are kept under, and the file as the index last saw it.
--
-- The size and the time are what say whether the recognition is of the bytes on
-- disk now. A file rewritten since is a file whose recognition was of other bytes,
-- and coordinates from it fall where those words no longer are.
SELECT COALESCE(producer, ''), COALESCE(hash, ''), size, modified_at
FROM sources
WHERE vault_id = ? AND path = ?;
