-- A vector for text no chunk holds any more.
--
-- Asked only where a source was cut again, so the text is known to have been
-- replaced rather than merely out of sight. A source that went from the vault
-- takes nothing with it: a folder that could not be read looks the same from
-- here as one whose files were deleted.
DELETE FROM vectors
WHERE fingerprint = unhex(?)
  AND NOT EXISTS (SELECT 1 FROM chunks WHERE hash = ?);
