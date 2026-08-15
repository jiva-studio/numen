-- Derived rows are replaced wholesale: diffing them against what was there
-- costs more than rewriting a handful of rows.
DELETE FROM headings WHERE vault_id = ? AND path = ?;
