-- Sources of one kind with no small chunk: the file changed, or it has never
-- been cut. Both need the same work.
--
-- A source that has only the chunk enclosing it needs the cut, because a small
-- chunk is what carries a vector. A source recorded under a recipe with no
-- content hash is a file with no text: it has nothing to cut.
SELECT path
FROM sources s
WHERE s.vault_id = ? AND s.kind = ?
  AND NOT (s.recipe IS NOT NULL AND s.content_hash IS NULL)
  AND NOT EXISTS (
      SELECT 1 FROM chunks c WHERE c.source_id = s.id AND c.parent_id IS NOT NULL
  )
ORDER BY path
LIMIT ?;
