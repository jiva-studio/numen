-- How far embedding has got is asked of the windows that carry vectors.
--
-- A large window carries none, so the total counts only the windows that do. The
-- index carries `parent`, so the question is one pass over an index.
CREATE INDEX chunks_by_window ON chunks (vault_id, parent, id);
