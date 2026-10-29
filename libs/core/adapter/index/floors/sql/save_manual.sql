-- The person's choice replaces itself and leaves the measurement alone.
INSERT INTO floors (vault, recipe, is_manual, manual_floor)
VALUES (?, ?, ?, ?)
ON CONFLICT (vault, recipe) DO UPDATE SET
    is_manual    = excluded.is_manual,
    manual_floor = excluded.manual_floor;
