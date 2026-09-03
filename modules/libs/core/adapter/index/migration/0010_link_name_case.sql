-- A name is compared without regard to case, so a note's extension comes off
-- the name a link is written by however it is spelled.
--
-- An index built before this holds `[[Entropy.MD]]` with the extension on, so
-- the link reaches nothing and the note it names lists no backlink. The names
-- are written again here out of the address beside them, which needs no file
-- read.

-- `rtrim(value, replace(value, '/', ''))` is the value up to and including its
-- last slash, so what follows it is the last segment. A value carrying no
-- slash keeps all of itself.
UPDATE links
SET value_base = substr(value, length(rtrim(value, replace(value, '/', ''))) + 1);

UPDATE links
SET value_base = substr(value_base, 1, length(value_base) - 3)
WHERE length(value_base) > 3 AND lower(substr(value_base, -3)) = '.md';
