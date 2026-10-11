-- The recipe whose vectors fill `chunks_vec`.
--
-- `chunks_vec` is one table for every vault, built for one width, so the recipe
-- is held once: the table has one row, and `id = 1` is what keeps it so. Two
-- recipes can share a width, which makes the width alone unable to say whose
-- bits the coarse rows hold. No row means no recipe filled the table, and the
-- next fit of the coarse index fills it again from the vectors already kept.
CREATE TABLE coarse_recipe (
    id     INTEGER PRIMARY KEY CHECK (id = 1),
    recipe TEXT NOT NULL
);
