-- What a note is: a note, a deck of cards, or the stencil a card is cut by.
-- The list is closed, and a note whose file says nothing is a note.
--
-- Every note already here was read before the key existed, so the default is
-- what each of them carries until its file is read again.
ALTER TABLE notes ADD COLUMN type TEXT NOT NULL DEFAULT 'note';

-- What every note says it is is read again.
--
-- A scan skips a file whose size and modification time still match what the
-- index holds, so a file that already said `type: deck` would keep the default
-- above until somebody edited it. The fingerprint of every note is emptied
-- here, and the next scan reads those files and writes what each one is. It
-- costs that scan one read of every note.
UPDATE sources SET size = -1, modified_at = -1 WHERE kind = 'note';

-- Which notes of a vault are of one kind. Nearly every note in a vault is a
-- note, and what this is asked is which of them are the few that are not.
CREATE INDEX notes_by_type ON notes (vault_id, type);
