INSERT INTO vaults (id, name, path)
VALUES (?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    path = excluded.path;
