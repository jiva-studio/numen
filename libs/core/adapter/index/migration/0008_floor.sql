-- The similarity floor of a vault under one embedding recipe.
--
-- A floor is a fact about a model and the text it was measured on, so the key is
-- the pair. A rebuild of the index reads the vault again and leaves this table as
-- it is: a calibration is bought with a model's time, and the person's own floor
-- is a choice.
--
-- `vault` is the identifier from the vault's own config file. `recipe` is the
-- value embeddingport.EmbeddingModel.Recipe gives.
--
-- `is_manual` marks `manual_floor` as in force. `floor` is what the last
-- calibration derived and is null before the first. `median`, `spread`,
-- `histogram` (a JSON array of counts), `query_count`, `passage_count` and
-- `calibrated_at` (seconds since the epoch) describe that calibration, and are
-- null with it.
CREATE TABLE floors (
    vault         TEXT NOT NULL,
    recipe        TEXT NOT NULL,
    is_manual     INTEGER NOT NULL DEFAULT 0,
    manual_floor  REAL NOT NULL DEFAULT 0,
    floor         REAL,
    median        REAL,
    spread        REAL,
    histogram     TEXT,
    query_count   INTEGER,
    passage_count INTEGER,
    calibrated_at INTEGER,
    PRIMARY KEY (vault, recipe)
) WITHOUT ROWID;
