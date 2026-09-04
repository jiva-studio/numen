-- `identifier` is the only column, and it is what the row already exists by,
-- so there is nothing left to write on a vault the index already has.
INSERT INTO vaults (identifier) VALUES (?) ON CONFLICT (identifier) DO NOTHING;
