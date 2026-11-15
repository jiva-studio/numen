-- Every event of one source, which its removal replaces.
DELETE FROM file_changes WHERE vault_id = ? AND source_id = ?;
