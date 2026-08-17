-- `hash` addresses the window's text, and is what the next cut of this source
-- compares against.
INSERT INTO chunks (source_id, vault_id, start, length, parent, location, hash)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id;
