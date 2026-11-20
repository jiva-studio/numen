INSERT INTO app_state (vault, key, value) VALUES (?, ?, ?)
ON CONFLICT DO UPDATE SET value = excluded.value
WHERE value <> excluded.value;
