DROP TABLE open_tabs;

CREATE TABLE app_state (
    vault TEXT,
    key   TEXT NOT NULL,
    value TEXT NOT NULL CHECK (json_valid(value))
);

CREATE UNIQUE INDEX app_state_by_vault_key ON app_state (vault, key) WHERE vault IS NOT NULL;
CREATE UNIQUE INDEX app_state_by_global_key ON app_state (key) WHERE vault IS NULL;
