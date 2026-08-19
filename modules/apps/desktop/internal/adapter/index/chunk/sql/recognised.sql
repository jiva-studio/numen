-- The sources of one kind whose text a producer made: the producer's name, and
-- the hash the files of that reading are kept under.
SELECT path, text_from, COALESCE(hash, '')
FROM sources
WHERE vault_id = ? AND kind = ? AND text_from IS NOT NULL
ORDER BY path;
