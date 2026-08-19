-- The names in one vault that match the words typed: a note's own title first,
-- then the headings inside notes.
--
-- Each half is ranked against its own population and the two are never weighed
-- against each other: a note called what was typed is a better answer than a
-- note with a line in it called that, whatever either score says. `kind` is
-- what holds them in that order.
--
-- `line` is -1 for a title, which stands on no line of the prose.
--
-- `at` is the name with the run that matched wrapped in the two marks bound to
-- the query, which is the index saying where it matched. A word reached by its
-- prefix is marked whole.
--
-- Each half is cut to its own limit before they meet, so a query matching the
-- whole vault ranks two limits' worth and not the vault.
SELECT * FROM (
    SELECT 0                                AS kind,
           s.path                           AS path,
           n.title                          AS title,
           -1                               AS line,
           highlight(titles_fts, 0, ?, ?)   AS at,
           bm25(titles_fts)                 AS score
    FROM titles_fts
    JOIN notes n ON n.source_id = titles_fts.rowid
    JOIN sources s ON s.id = n.source_id
    WHERE titles_fts MATCH ? AND n.vault_id = ?
    ORDER BY score
    LIMIT ?
)
UNION ALL
SELECT * FROM (
    SELECT 1                                AS kind,
           s.path                           AS path,
           n.title                          AS title,
           h.line                           AS line,
           highlight(headings_fts, 0, ?, ?) AS at,
           bm25(headings_fts)               AS score
    FROM headings_fts
    JOIN headings h ON h.id = headings_fts.rowid
    JOIN notes n ON n.source_id = h.note_id
    JOIN sources s ON s.id = n.source_id
    WHERE headings_fts MATCH ? AND n.vault_id = ?
    ORDER BY score
    LIMIT ?
)
ORDER BY kind, score;
