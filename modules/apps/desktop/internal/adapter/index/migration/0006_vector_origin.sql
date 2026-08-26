-- A vector is kept under where it was made as well as under what made it: the
-- placement that filled the index, which is a model run on this machine or a
-- service at one address.
--
-- One name is run here and served by more than one place, and the numbers those
-- give for one text are not the same numbers. Every vector here was kept under
-- a key that does not say which of them made it, so they go and the next pass
-- over a vault makes them again.
DELETE FROM vectors;

-- The coarse form of a vector is a reading of the vector, and the two are one
-- value.
DELETE FROM chunks_vec;
