-- The row number of one source of one kind. A path naming a file of another
-- kind answers nothing, so a repository for one kind cannot reach another's.
SELECT id FROM sources WHERE vault_id = ? AND kind = ? AND path = ?;
