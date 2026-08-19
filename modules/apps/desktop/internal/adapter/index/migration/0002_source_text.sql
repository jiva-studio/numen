-- Where a source's text is read from, when it is not the source's own file.
--
-- A scanned document holds no text a machine can take out of it. What reading
-- it produced is a file of its own, and this source's chunks are places in that
-- file: showing a passage reads it, and not the document.
--
-- Null is the ordinary case, and the only case for a note or a book whose text
-- is its own: the text of a source is the source.
--
-- What is stored is a name in the application's own store, never a path in the
-- vault. The folder that store lives in is a setting, and a row that wrote the
-- folder's name down would stop finding its own files the day it changed.
--
-- It is cleared by the same write that clears `hash` and `recipe`, because it is
-- the same fact: the file is not the file that was read. A row keeping it across
-- a change would answer searches with text the document no longer holds, and
-- nothing would say so.
ALTER TABLE sources ADD COLUMN text_path TEXT;

-- Which sources of a vault stand on a file of their own. It is asked once a
-- scan, to find the ones whose file a person deleted by hand, and it is answered
-- in proportion to the documents that were read rather than to the library.
CREATE INDEX sources_by_text ON sources (vault_id, kind) WHERE text_path IS NOT NULL;
