-- The sources of one kind whose text a producer made: the producer's name, and
-- the hash the files of that reading are kept under.
SELECT path, producer, COALESCE(hash, '')
FROM sources
WHERE vault_id = ? AND kind = ? AND producer IS NOT NULL
ORDER BY path;
