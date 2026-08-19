-- How far reading a vault for meaning has got: small windows held, and how many
-- of them carry a vector under the recipe now in use.
--
-- Only the windows that carry vectors are counted. A large window is text to
-- read once something has been found, and asks for nothing.
--
-- One pass over an index, joining each window to its vector by the text it
-- holds. Counting a nullable column counts the rows where it is present, which
-- is what "has a vector under this recipe" means.
SELECT count(*), count(v.fingerprint)
  FROM chunks c
  LEFT JOIN vectors v ON v.fingerprint = unhex(c.hash) AND v.recipe = ?
 WHERE c.vault_id = ? AND c.parent IS NOT NULL;
