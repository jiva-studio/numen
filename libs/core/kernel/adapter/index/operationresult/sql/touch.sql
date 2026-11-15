-- Marks the stored outputs of the input hashes ?3 used at ?4.
UPDATE operation_results
SET used_at = ?4
WHERE operation = ?1
  AND recipe_hash = ?2
  AND input_hash IN (SELECT value FROM json_each(?3));
