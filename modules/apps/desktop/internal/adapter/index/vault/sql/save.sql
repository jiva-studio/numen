INSERT INTO vaults (identifier, name, path)
VALUES (?, ?, ?)
ON CONFLICT (identifier) DO UPDATE SET
    name = excluded.name,
    path = excluded.path;
