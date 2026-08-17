-- The coarse pass: the nearest chunks in one vault.
--
-- The vault is constrained inside the query: a nearest-neighbour search answers
-- with the whole table's best k.
SELECT chunk_id, distance
FROM chunks_vec
WHERE embedding MATCH vec_bit(?)
  AND vault_id = ?
  AND k = ?
ORDER BY distance;
