-- The marker a link record draws at the end that points at its target.
--
-- `marker` is null for a record that draws nothing. The only value written today
-- is 'arrow'. No file written before this migration carries a marker, so a row
-- that exists already is correct with a null.
ALTER TABLE links ADD COLUMN marker TEXT;
