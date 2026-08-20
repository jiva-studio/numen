-- The names of the parts a source divides into, so that a section is something
-- a person can find and not only a label an answer carries.
--
-- The words half ranks a chunk by how often the words appear in it, and a
-- chapter's opening paragraph says its subject once — in the heading — while a
-- paragraph in the middle of it says it four times. Asked where a book speaks
-- about a thing, the index answered with the densest paragraph and not with the
-- chapter. What a person asked for was the chapter.
--
-- The rowid is the chunk the part opens. A section answers with the chunk that
-- begins where it begins, so a hit on a name is a passage standing at the start
-- of the section, and everything that reads a passage back reads this one the
-- same way. No second table: where a part is, is where its chunk is.
--
-- A chunk that opens a part and a subsection under it carries both names, which
-- is one row holding two lines.
--
-- `content=''` keeps no copy: nothing reads a name back. What an answer shows is
-- the chunk's own location, which already says what the section is called.
CREATE VIRTUAL TABLE parts_fts USING fts5 (
    text,
    content='',
    contentless_delete=1
);

-- Every source already cut was cut without them. The names of the parts are not
-- in the text and not in the offsets, so nothing about a source says they are
-- missing and no scan would notice.
--
-- Owing its text is how a source asks to be cut again. What it was read from
-- stays: a recognition is an hour of a machine, and cutting is reading files
-- that are on disk. A window that says the same thing keeps its row, so the
-- vectors are kept with it.
UPDATE sources SET recipe = NULL;
