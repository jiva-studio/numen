-- The coarse pass over the sources whose path starts with the prefix: the
-- chunks of one vault under the prefix nearest the query by Hamming distance.
--
-- The prefix is applied before the limit, so every chunk under it is compared,
-- however many nearer chunks are outside it. The work grows with the chunks
-- under the prefix, not with the vault. The prefix is given in its composed
-- (NFC) form ?3 and its decomposed (NFD) form ?6, and a path that starts with
-- either is under it. Each form is a range of the sources path index: every
-- path that starts with ?3 sorts from ?3 up to the bound ?5, and every path that
-- starts with ?6 from ?6 up to the bound ?7; a bound is the least value above
-- all of them by bytes, which the caller computes. The comparison of the start
-- of the path keeps exactly those.
SELECT v.chunk_id
FROM sources s
JOIN chunks c ON c.source_id = s.id
JOIN chunks_vec v ON v.chunk_id = c.id
WHERE s.id IN (
    SELECT id FROM sources
    WHERE vault_id = ?2 AND path >= ?3 AND path < ?5 AND substr(path, 1, length(?3)) = ?3
    UNION
    SELECT id FROM sources
    WHERE vault_id = ?2 AND path >= ?6 AND path < ?7 AND substr(path, 1, length(?6)) = ?6
)
ORDER BY vec_distance_hamming(v.coarse, vec_bit(?1)), v.chunk_id
LIMIT ?4;
