-- How far reading a vault for meaning has got: small windows held, and how many
-- of them carry a vector for the model now in use.
--
-- Only the windows that carry vectors are counted. A large window is text to
-- read once something has been found, and asks for nothing.
--
-- One pass over an index, joining each window to its vector by row number.
-- Counting a nullable column counts the rows where it is present, which is what
-- "has a vector for this model" means.
SELECT count(*), count(v.chunk_id)
  FROM chunks c
  LEFT JOIN vectors v ON v.chunk_id = c.id AND v.model = ?
 WHERE c.vault_id = ? AND c.parent IS NOT NULL;
