-- The small chunks of one vault that carry a vector under the recipe, in the
-- order of their content hashes.
--
-- A large chunk carries no vector and is left out by asking for the ones that sit
-- inside something.
SELECT c.source_id, e.embedding
FROM chunks c
JOIN vaults v ON v.id = c.vault_id
JOIN vectors e ON e.hash = unhex(c.hash) AND e.recipe = ?2
WHERE v.identifier = ?1 AND c.parent_id IS NOT NULL
ORDER BY c.hash, c.id
LIMIT ?3;
