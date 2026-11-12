-- The versions of whatever a move lands on, apart from the ones that move.
-- Arguments: vault, the target, the range under the target, the source, the
-- range under the source.
DELETE FROM text_versions
WHERE vault = ?
  AND (path = ? OR (path >= ? AND path < ?))
  AND NOT (path = ? OR (path >= ? AND path < ?));
