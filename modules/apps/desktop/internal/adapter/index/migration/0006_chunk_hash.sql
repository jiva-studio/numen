-- The address a chunk's text gives it, and what identifies the chunk. Cutting a
-- source again is a comparison against this column: a window whose hash is
-- already on a row keeps that row, and its vector and its full-text row with it.
--
-- A row written before this column existed carries the empty string, which no
-- text hashes to, so such a row is replaced the next time its source is cut.
ALTER TABLE chunks ADD COLUMN hash TEXT NOT NULL DEFAULT '';
