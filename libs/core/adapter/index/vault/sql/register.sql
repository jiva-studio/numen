-- A vault the index already has a row for keeps it.
INSERT INTO vaults (identifier) VALUES (?) ON CONFLICT (identifier) DO NOTHING;
