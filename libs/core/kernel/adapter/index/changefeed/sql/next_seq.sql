-- The number of the next change of the feed of a vault.
UPDATE change_feeds SET last_seq = last_seq + 1 WHERE vault_id = ? RETURNING last_seq;
