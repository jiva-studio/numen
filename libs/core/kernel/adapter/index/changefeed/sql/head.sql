-- The feed of a vault, by the identity the vault has outside this database.
SELECT f.feed_id, f.last_seq, f.trimmed_seq
FROM change_feeds f
JOIN vaults v ON v.id = f.vault_id
WHERE v.identifier = ?;
