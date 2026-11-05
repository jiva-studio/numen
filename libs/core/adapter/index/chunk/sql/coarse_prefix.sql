-- The coarse pass over the sources whose path starts with the prefix: the
-- chunks of one vault under the prefix nearest the query by Hamming distance.
--
-- The prefix is applied before the limit, so every chunk under it is compared,
-- however many nearer chunks lie outside it. The work grows with the chunks
-- under the prefix, not with the vault. The range on the path reaches the
-- sources through the index of their paths: every path that starts with the
-- prefix sorts from the prefix up to the prefix followed by the largest
-- character, and the comparison of the start of the path keeps exactly those.
SELECT v.chunk_id
FROM sources s
JOIN chunks c ON c.source_id = s.id
JOIN chunks_vec v ON v.chunk_id = c.id
WHERE s.vault_id = ?2
  AND s.path >= ?3
  AND s.path < ?3 || char(1114111)
  AND substr(s.path, 1, length(?3)) = ?3
ORDER BY vec_distance_hamming(v.coarse, vec_bit(?1)), v.chunk_id
LIMIT ?4;
