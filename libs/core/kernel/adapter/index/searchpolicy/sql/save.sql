INSERT INTO search_floors (vault, recipe, floor)
VALUES (?, ?, ?)
ON CONFLICT (vault, recipe) DO UPDATE SET floor = excluded.floor;
