-- What the vault held at one path, and everything under it, filed where it now
-- is. A folder and all it holds travel in this one statement.
--
-- Each row keeps what its path holds past the folder that moved, so the file at
-- the path itself keeps nothing and lands on the destination.
--
-- The path is one lookup, and what is under it is the range from the folder's
-- slash to the byte after it.
UPDATE sources
SET path = ? || substr(path, ?)
WHERE vault_id = ?
  AND (path = ? OR (path >= ? AND path < ?));
