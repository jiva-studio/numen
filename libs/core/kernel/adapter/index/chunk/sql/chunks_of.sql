-- Every chunk of one source: its number, what its text hashed to, and whether
-- something encloses it. This is what a fresh cut of the source is compared
-- against.
SELECT id, hash, parent_id IS NOT NULL FROM chunks WHERE source_id = ?;
