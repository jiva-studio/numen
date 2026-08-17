-- Sources of one kind with no small window: the file changed, or it has never
-- been cut. Both owe the same work.
--
-- A source that has only the window enclosing it owes the cut, because a small
-- window is what carries a vector.
SELECT path
FROM sources s
WHERE s.vault_id = ? AND s.kind = ?
  AND NOT EXISTS (
      SELECT 1 FROM chunks c WHERE c.source_id = s.id AND c.parent IS NOT NULL
  )
ORDER BY path
LIMIT ?;
