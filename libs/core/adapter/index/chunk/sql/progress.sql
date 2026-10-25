-- How far indexing a vault for meaning has got: small chunks held, and how many
-- of them owe nothing under the recipe now in use.
--
-- Only the chunks that carry vectors are counted. A large chunk is text to
-- read once something has been found, and asks for nothing.
--
-- A chunk owes nothing when the recipe holds a vector for its text and the
-- coarse pass holds a row for the chunk, which is what the question of what is
-- owed asks.
SELECT count(*),
       count(CASE WHEN v.hash IS NOT NULL
                   AND EXISTS (SELECT 1 FROM chunks_vec WHERE chunk_id = c.id) THEN 1 END)
  FROM chunks c
  LEFT JOIN vectors v ON v.hash = unhex(c.hash) AND v.recipe = ?
 WHERE c.vault_id = ? AND c.parent_id IS NOT NULL;
