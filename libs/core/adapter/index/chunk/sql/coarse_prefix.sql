-- The coarse pass over the sources whose path starts with the prefix: the
-- chunks of one vault under the prefix nearest the query by Hamming distance.
--
-- The prefix is applied before the limit, so every chunk under it is compared,
-- however many nearer chunks lie outside it. The work grows with the chunks
-- under the prefix, not with the vault. The prefix is compared as text.
SELECT v.chunk_id
FROM sources s
JOIN chunks c ON c.source_id = s.id
JOIN chunks_vec v ON v.chunk_id = c.id
WHERE s.vault_id = ?2
  AND substr(s.path, 1, length(?3)) = ?3
ORDER BY vec_distance_hamming(v.coarse, vec_bit(?1)), v.chunk_id
LIMIT ?4;
