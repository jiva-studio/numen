-- Every note filed under one name, whatever folder it sits in.
--
-- The basename column holds the folded name, so a vault holding `Entropy.md`
-- and `entropy.md` is a vault where one name means two notes — which is the
-- answer this question exists to give. The name asked for is folded too.
SELECT s.path
FROM notes n
JOIN sources s ON s.id = n.source_id
WHERE n.vault_id = ? AND n.basename = ?
ORDER BY 1;
