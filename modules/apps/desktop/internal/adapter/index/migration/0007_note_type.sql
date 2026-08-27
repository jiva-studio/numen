-- What a note is: a note, a deck of cards, or the stencil a card is cut by.
-- The list is closed, and a note whose file says nothing is a note.
--
-- Every note already here was read before the key existed, and every one of
-- them is a note. The default writes that in, and a file carrying the key is
-- read again the next time it changes.
ALTER TABLE notes ADD COLUMN type TEXT NOT NULL DEFAULT 'note';

-- Which notes of a vault are of one kind. Nearly every note in a vault is a
-- note, and what this is asked is which of them are the few that are not.
CREATE INDEX notes_by_type ON notes (vault_id, type);
