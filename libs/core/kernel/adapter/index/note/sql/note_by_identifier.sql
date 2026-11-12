-- Not scoped to one vault. An identifier names one note in the world, so a link
-- written that way resolves wherever that note is, and the vault it landed in
-- comes back with it.
--
-- Ordered because a copied note file carries a copied identifier, and two notes
-- claiming one identity must resolve the same way on every machine and every
-- run. Which of them wins is arbitrary; that it is always the same one is not.
SELECT v.identifier, s.path
FROM notes n
JOIN sources s ON s.id = n.source_id
JOIN vaults v ON v.id = n.vault_id
WHERE n.identifier = ?
ORDER BY v.identifier, s.path
LIMIT 1;
