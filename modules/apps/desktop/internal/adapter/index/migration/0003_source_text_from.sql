-- Which producer made the text this source's chunks are places in.
--
-- A scanned document holds no text a machine can take out of it. What reading
-- it produced is a file of its own, and this source's chunks are places in that
-- file: showing a passage reads it, and not the document.
--
-- Null is the ordinary case, and the only case for a note or a book whose text
-- is its own: the text of a source is the source.
--
-- The four files one recognition writes are all composed from the producer and
-- the hash, which is already a column here, so a name is composed where it is
-- needed and nothing takes an extension off a stored string.
--
-- It is cleared by the same write that clears `hash` and `recipe`, because it is
-- the same fact: the file is not the file that was read. A row keeping it across
-- a change would answer searches with text the document no longer holds, and
-- nothing would say so.
ALTER TABLE sources RENAME COLUMN text_path TO text_from;

-- The column held a whole name before it held a producer. The producer is what
-- stands in front of the first separator, and a row keeping the whole name
-- composes a name from it and finds nothing under it.
UPDATE sources
SET text_from = substr(text_from, 1, instr(text_from, '/') - 1)
WHERE text_from LIKE '%/%';

-- Which sources of a vault stand on a text a producer made. It is asked once a
-- scan, to find the ones whose file a person deleted by hand, and it is answered
-- in proportion to the documents that were read rather than to the library.
DROP INDEX sources_by_text;

CREATE INDEX sources_by_text ON sources (vault_id, kind) WHERE text_from IS NOT NULL;
