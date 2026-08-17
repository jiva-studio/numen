INSERT INTO chunks (source_id, vault_id, start, length, parent, location)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;
