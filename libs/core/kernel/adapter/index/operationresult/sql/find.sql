-- The stored outputs of the input hashes ?3, a JSON array, under the operation ?1 and the recipe ?2.
SELECT r.input_hash, r.output
FROM operation_results r
WHERE r.operation = ?1
  AND r.recipe_hash = ?2
  AND r.input_hash IN (SELECT value FROM json_each(?3));
