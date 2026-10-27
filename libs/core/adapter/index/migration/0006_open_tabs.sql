-- The tabs the window last had open on each vault, kept across restarts.
--
-- A rebuild of the index reads the vault again and leaves this table as it is:
-- the open tabs are window state, not something read from the files.
--
-- `vault` is the identifier from the vault's own config file. `position` is the
-- order the window wrote the tabs in. `state` is the JSON the plugin owning the
-- kind saved, stored unread. `is_front` marks the tab in front.
CREATE TABLE open_tabs (
    vault    TEXT NOT NULL,
    position INTEGER NOT NULL,
    id       TEXT NOT NULL,
    kind     TEXT NOT NULL,
    path     TEXT NOT NULL,
    title    TEXT NOT NULL,
    state    TEXT NOT NULL,
    is_front INTEGER NOT NULL,
    PRIMARY KEY (vault, position)
);
