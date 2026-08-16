-- Every note filed under one name, whatever folder it sits in.
--
-- The basename column folds case, so a vault holding `Entropy.md` and
-- `entropy.md` is a vault where one name means two notes — which is the answer
-- this question exists to give.
SELECT path FROM notes WHERE vault_id = ? AND basename = ? ORDER BY 1;
