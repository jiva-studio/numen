-- What a model made, addressed by the text it read and the recipe it read it
-- under.
--
-- Everything else this index holds is a reading of files that are still on
-- disk, and a scan puts it back for the cost of reading them. A vector is
-- bought: minutes of a machine, or money and a network. It is kept where no
-- renumbering of chunks and no rebuilding of the index can reach it.
--
-- The recipe names everything that decides what the vector is: where it was
-- made, which model, how wide, where the text was cut off, and how the numbers
-- are stored. Change any of them and the old rows are simply not found, still
-- here, still there if the setting goes back.
ALTER TABLE vectors RENAME TO chunk_vectors;

CREATE TABLE vectors (
    fingerprint BLOB NOT NULL,
    recipe      TEXT NOT NULL,
    v           BLOB NOT NULL
);

-- A rowid table with the key in an index of its own: the vector is a kilobyte,
-- and a key that carries it is a key every probe reads a kilobyte to answer.
CREATE UNIQUE INDEX vectors_of ON vectors (fingerprint, recipe);

-- What this index has already bought, kept. A chunk cut before its text was
-- fingerprinted carries none, and its vector waits in `chunk_vectors` until the
-- source is cut again.
INSERT OR IGNORE INTO vectors (fingerprint, recipe, v)
SELECT unhex(c.hash), cv.model || '/' || cv.kind, cv.v
FROM chunk_vectors cv
JOIN chunks c ON c.id = cv.chunk_id
WHERE length(c.hash) = 64;

-- Finding a chunk by the text it holds, which is how a vector is claimed.
CREATE INDEX chunks_by_hash ON chunks (hash);
