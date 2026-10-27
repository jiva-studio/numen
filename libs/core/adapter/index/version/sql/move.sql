-- The versions of a file, or of everything under a folder, filed at the path
-- they now have. Each row keeps what its path holds past the folder that moved.
UPDATE text_versions
SET path = ? || substr(path, ?)
WHERE vault = ?
  AND (path = ? OR (path >= ? AND path < ?));
