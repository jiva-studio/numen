-- The results of connection operations, by operation, recipe and input, so that a query that asks the same
-- of the same service again reuses the answer. used_at is nanoseconds since the epoch.
CREATE TABLE operation_results (
    operation   TEXT    NOT NULL,
    recipe_hash TEXT    NOT NULL,
    input_hash  TEXT    NOT NULL,
    output      BLOB    NOT NULL,
    used_at     INTEGER NOT NULL,
    PRIMARY KEY (operation, recipe_hash, input_hash)
) WITHOUT ROWID;

CREATE INDEX operation_results_used_at ON operation_results (used_at);
