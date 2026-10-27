-- Every source that is not a note is of one kind, `file`: the core no longer
-- tells one file format from another. A row of a kind the core no longer walks
-- is one no scan would update or remove.
UPDATE sources SET kind = 'file' WHERE kind <> 'note';
