-- The name and the path are the list's, and a vault the index already knows
-- keeps the copy of them it was given.
INSERT INTO vaults (identifier, name, path)
VALUES (?, ?, ?)
ON CONFLICT (identifier) DO NOTHING;
