-- Those of the paths, given as a JSON array, that are sources of the vault.
SELECT path FROM sources
WHERE vault_id = ? AND path IN (SELECT value FROM json_each(?))
ORDER BY path;
