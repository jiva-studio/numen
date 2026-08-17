-- How far embedding has got is asked of the windows that carry vectors.
--
-- A chunk that encloses others carries none, so the total counts only the ones
-- that do. The index carries `parent`, so the question is one pass over an index.
CREATE INDEX chunks_by_vault_parent ON chunks (vault_id, parent, id);
