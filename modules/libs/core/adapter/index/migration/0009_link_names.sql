-- The name a link is written by is its last segment, and only a note's
-- extension comes off it. A dot inside the name belongs to the name:
-- `[[Lecture 1.2]]` names a note filed under all of it.
--
-- An index built before this holds those names cut at their last dot, so the
-- link reaches nothing and the note it names lists no backlink. They are
-- written again here out of the address beside them, which needs no file read.

-- `rtrim(value, replace(value, '/', ''))` is the value up to and including its
-- last slash, so what follows it is the last segment. A value carrying no
-- slash keeps all of itself.
UPDATE links
SET value_base = substr(value, length(rtrim(value, replace(value, '/', ''))) + 1);

UPDATE links
SET value_base = substr(value_base, 1, length(value_base) - 3)
WHERE length(value_base) > 3 AND substr(value_base, -3) = '.md';
