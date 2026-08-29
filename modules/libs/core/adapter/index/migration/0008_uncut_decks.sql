-- A deck and a stencil are not searched by their text. Neither is cut, so
-- neither holds a chunk and neither holds a vector, and the only headings kept
-- are the ones a person wrote: a deck's sections and its cards, and none of a
-- stencil's.
--
-- An index built before this holds all three for those notes. They come out
-- here, and the next scan reads the files again.
--
-- An index that has not yet learnt what a note is arrives here in the same open
-- as 0007, with every type still the default that migration adds, so nothing
-- below matches. 0007 empties the fingerprint of every note, and the scan that
-- follows writes each file's type and its outline together.

-- The full-text index, the names of the parts and the vector index are all
-- addressed by the chunk's own number. They run before the chunks go, which is
-- where the numbers are read from.
DELETE FROM chunks_fts WHERE rowid IN (
    SELECT chunks.id FROM chunks
    JOIN notes ON notes.source_id = chunks.source_id
    WHERE notes.type IN ('deck', 'stencil')
);

DELETE FROM parts_fts WHERE rowid IN (
    SELECT chunks.id FROM chunks
    JOIN notes ON notes.source_id = chunks.source_id
    WHERE notes.type IN ('deck', 'stencil')
);

DELETE FROM chunks_vec WHERE chunk_id IN (
    SELECT chunks.id FROM chunks
    JOIN notes ON notes.source_id = chunks.source_id
    WHERE notes.type IN ('deck', 'stencil')
);

-- A vector is addressed by the text it was made for, never by a chunk, so one
-- goes only where every chunk holding that text is a deck's or a stencil's.
-- Two notes writing the same words share the row, and the other note keeps it.
DELETE FROM vectors
WHERE EXISTS (
    SELECT 1 FROM chunks
    JOIN notes ON notes.source_id = chunks.source_id
    WHERE unhex(chunks.hash) = vectors.fingerprint
      AND notes.type IN ('deck', 'stencil')
  )
  AND NOT EXISTS (
    SELECT 1 FROM chunks
    LEFT JOIN notes ON notes.source_id = chunks.source_id
    WHERE unhex(chunks.hash) = vectors.fingerprint
      AND (notes.type IS NULL OR notes.type NOT IN ('deck', 'stencil'))
  );

DELETE FROM chunks WHERE source_id IN (
    SELECT source_id FROM notes WHERE type IN ('deck', 'stencil')
);

-- The index of headings is addressed by the heading's own number, and runs
-- before the headings go. A deck's sections and its cards are written again by
-- the scan below, without the field names and without the marks.
DELETE FROM headings_fts WHERE rowid IN (
    SELECT headings.id FROM headings
    JOIN notes ON notes.source_id = headings.note_id
    WHERE notes.type IN ('deck', 'stencil')
);

DELETE FROM headings WHERE note_id IN (
    SELECT source_id FROM notes WHERE type IN ('deck', 'stencil')
);

-- Every deck and every stencil is read again.
--
-- A scan skips a file whose size and modification time still match what the
-- index holds, so the headings emptied above would stay empty until somebody
-- edited the file. The fingerprint of each of those files is emptied here, and
-- it costs that scan one read of each.
UPDATE sources SET size = -1, modified_at = -1
WHERE id IN (SELECT source_id FROM notes WHERE type IN ('deck', 'stencil'));
