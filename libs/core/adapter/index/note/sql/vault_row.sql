-- The vault's identity inside this database, from the one it carries in the
-- world.
SELECT id FROM vaults WHERE identifier = ?;
