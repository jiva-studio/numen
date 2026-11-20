CREATE TABLE app_state (
    vault TEXT,
    key   TEXT NOT NULL,
    value TEXT NOT NULL CHECK (json_valid(value))
);

CREATE UNIQUE INDEX app_state_by_vault_key ON app_state (vault, key) WHERE vault IS NOT NULL;
CREATE UNIQUE INDEX app_state_by_global_key ON app_state (key) WHERE vault IS NULL;

WITH workspaces AS (
    SELECT vault, json_group_array(json_object(
        'id', id, 'kind', kind, 'path', path, 'title', title, 'state', state
    ) ORDER BY position) AS tabs
    FROM open_tabs
    GROUP BY vault
)
INSERT INTO app_state (vault, key, value)
SELECT vault, 'editor.workspace', json_object(
    'tabs', json(tabs),
    'activeId', coalesce((
        SELECT id FROM open_tabs t
        WHERE t.vault = w.vault AND t.is_front
        ORDER BY t.position DESC
        LIMIT 1
    ), '')
)
FROM workspaces w;

DROP TABLE open_tabs;
