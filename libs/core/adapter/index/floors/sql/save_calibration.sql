-- A calibration replaces the measurement and leaves the person's choice alone.
INSERT INTO floors (vault, recipe, floor, median, spread, histogram, query_count, passage_count, calibrated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault, recipe) DO UPDATE SET
    floor         = excluded.floor,
    median        = excluded.median,
    spread        = excluded.spread,
    histogram     = excluded.histogram,
    query_count   = excluded.query_count,
    passage_count = excluded.passage_count,
    calibrated_at = excluded.calibrated_at;
