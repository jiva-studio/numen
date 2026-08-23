-- A vector is kept under what decides what it is: the model, its width, where
-- a text is cut off, how the model's output becomes one vector, and how the
-- numbers are stored.
--
-- Where the vector was made is no longer among them. One model runs on this
-- machine and behind a service, and a vault filled by the one is asked by the
-- other, so an address in the key put the two out of reach of each other.
--
-- How the model is pooled is now among them. Every vector already here was
-- made by mean pooling, which is the only pooling that existed when they were
-- made, so the word is written in rather than the rows being dropped: a vector
-- is bought with minutes of a machine or with money, and this file is the
-- difference between an upgrade and a bill.
UPDATE vectors
SET recipe = replace(substr(recipe, instr(recipe, '|') + 1), '|int8', '|mean|int8')
WHERE instr(recipe, '|') > 0;
