-- The source of a vault at a path, and what tells whether its content changed.
SELECT id, size, modified_at, coalesce(content_hash, '') FROM sources WHERE vault_id = ? AND path = ?;
