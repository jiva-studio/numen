-- The titles of the notes of one vault, in the order of their content hashes.
--
-- A note's source carries a hash once something has computed it; a note without
-- one is placed by the hash of its first chunk. The order is a function of the
-- content, so one index yields one sample.
SELECT s.id, n.title
FROM notes n
JOIN sources s ON s.id = n.source_id
JOIN vaults v ON v.id = s.vault_id
WHERE v.identifier = ?1 AND TRIM(n.title) <> ''
ORDER BY COALESCE(s.hash, (SELECT MIN(c.hash) FROM chunks c WHERE c.source_id = s.id), ''), s.id
LIMIT ?2;
