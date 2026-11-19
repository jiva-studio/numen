-- Not scoped to one vault. Each identifier names one note in the world, so a link
-- written that way resolves wherever that note is, and the vault it landed in
-- comes back with it.
--
-- Ordered because a copied note file carries a copied identifier, and two notes
-- claiming one identity must resolve the same way on every machine and every
-- run.
SELECT n.identifier, v.identifier, s.path
FROM notes n
JOIN sources s ON s.id = n.source_id
JOIN vaults v ON v.id = n.vault_id
WHERE n.identifier IN (SELECT value FROM json_each(?))
ORDER BY n.identifier, v.identifier, s.path;
