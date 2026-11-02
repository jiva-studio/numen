-- The effective similarity floor of a vault under one embedding recipe.
--
-- A search by meaning applies this floor when its query names none. A plugin
-- measures or chooses the floor and stores it. The core keeps one value for each
-- pair of vault and recipe and holds no statistics of how it was found. A rebuild
-- of the index reads the vault again and leaves this table as it is.
--
-- `vault` is the identifier from the vault's own config file. `recipe` is the
-- value embeddingport.EmbeddingModel.Recipe gives. `floor` is a cosine
-- similarity above zero and at most one.
--
-- The calibration records of table `floors` are not carried over. The plugin
-- that owns calibration keeps its own records, and a vault is calibrated again.
DROP TABLE floors;

CREATE TABLE search_floors (
    vault  TEXT NOT NULL,
    recipe TEXT NOT NULL,
    floor  REAL NOT NULL,
    PRIMARY KEY (vault, recipe)
) WITHOUT ROWID;
