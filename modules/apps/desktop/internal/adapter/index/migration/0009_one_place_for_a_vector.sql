-- A vector is kept in one place: where the text it was made from addresses it.
--
-- What a chunk carries is read from the text that chunk holds, so a second copy
-- beside the row number says nothing the first does not, and costs a kilobyte a
-- chunk to say it.
DROP TABLE chunk_vectors;
