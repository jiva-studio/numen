-- Where a link note points, in the one form every spelling of it reaches. It is
-- null on every other note.
--
-- What was fetched from an address is kept under a name made from the address,
-- so the question of whether anything still points at what is on disk is asked
-- of this column. A vault's notes are read again by the next scan, which is
-- what fills it.
ALTER TABLE notes ADD COLUMN address TEXT;

-- Which notes point at one address. A sweep asks it of every artifact it holds,
-- and two notes that pasted one video answer together.
CREATE INDEX notes_by_address ON notes (address);
