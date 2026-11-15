INSERT INTO operation_results (operation, recipe_hash, input_hash, output, used_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (operation, recipe_hash, input_hash) DO UPDATE SET output = excluded.output, used_at = excluded.used_at;
