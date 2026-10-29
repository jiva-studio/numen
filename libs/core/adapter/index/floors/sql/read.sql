SELECT is_manual, manual_floor, floor, median, spread, histogram, query_count, passage_count, calibrated_at
FROM floors
WHERE vault = ? AND recipe = ?;
