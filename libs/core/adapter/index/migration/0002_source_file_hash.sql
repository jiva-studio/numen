-- The file a source's hash was taken from, and when: the inode, where the file
-- system gives one, and the time the hash was recorded, in nanoseconds since
-- the epoch. NULL in either is a hash that is checked by content once.
ALTER TABLE sources ADD COLUMN file_id INTEGER;
ALTER TABLE sources ADD COLUMN hashed_at INTEGER;
