-- Removes the rows that used_at, then the key, put first, as many as the table holds above ?1.
DELETE FROM operation_results
WHERE (operation, recipe_hash, input_hash) IN (
    SELECT operation, recipe_hash, input_hash
    FROM operation_results
    ORDER BY used_at, operation, recipe_hash, input_hash
    LIMIT max((SELECT count(*) FROM operation_results) - ?1, 0)
);
