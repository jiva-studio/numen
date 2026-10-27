-- The text versions of the files of every vault: which plugin made each text,
-- from what, and the file in that plugin's folder that holds it.
--
-- A rebuild of the index reads the vault again and leaves this table as it is:
-- what a version was made from cannot be read back from the files.
--
-- `vault` is the identifier from the vault's own config file, so a version
-- outlives the vault's row in `vaults`. `id` is the order versions were first
-- recorded in. A version recorded again keeps its row.
CREATE TABLE text_versions (
    id                INTEGER PRIMARY KEY,
    vault             TEXT NOT NULL,
    path              TEXT NOT NULL,
    plugin_id         TEXT NOT NULL,
    key               TEXT NOT NULL,
    source_version_id TEXT NOT NULL,
    kind              TEXT NOT NULL,
    file              TEXT NOT NULL,
    source_hash       TEXT NOT NULL,
    hash              TEXT NOT NULL,
    create_time       INTEGER NOT NULL,
    UNIQUE (vault, path, plugin_id, key)
);

CREATE INDEX text_versions_by_plugin ON text_versions (vault, plugin_id);
