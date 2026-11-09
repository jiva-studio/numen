-- What the index believes about every file of one kind in a vault.
SELECT path, size, modified_at FROM sources WHERE vault_id = ? AND kind = ?;
