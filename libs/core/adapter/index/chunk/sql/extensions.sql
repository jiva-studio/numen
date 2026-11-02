-- The sources of a vault by file extension, and the small chunks of each.
--
-- The extension is what follows the last dot of the path, in lower case. A path with no dot, or
-- with a slash after its last dot (a dot in a folder name), has none. Only the chunks that carry
-- vectors are counted, as progress.sql counts them.
--
-- The result is one row for each extension, and the extensions are few.
WITH named AS (
    SELECT s.id,
           CASE WHEN instr(s.path, '.') = 0 THEN ''
                ELSE lower(replace(s.path, rtrim(s.path, replace(s.path, '.', '')), ''))
           END AS ext
      FROM sources s
     WHERE s.vault_id = ?
), plain AS (
    SELECT id,
           CASE WHEN instr(ext, '/') > 0 OR instr(ext, char(92)) > 0 THEN '' ELSE ext END AS ext
      FROM named
)
SELECT p.ext, count(DISTINCT p.id), count(c.id)
  FROM plain p
  LEFT JOIN chunks c ON c.source_id = p.id AND c.vault_id = ? AND c.parent_id IS NOT NULL AND c.length >= ?
 GROUP BY p.ext
 ORDER BY 2 DESC, 1;
