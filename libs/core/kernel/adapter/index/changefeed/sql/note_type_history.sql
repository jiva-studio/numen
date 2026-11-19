-- The last sequence that affected each note type before the source was removed.
SELECT json_group_object(key, last_seq)
FROM (
    SELECT t.key, max(t.value) AS last_seq
    FROM file_changes c, json_each(c.note_types) t
    WHERE c.vault_id = ? AND c.source_id = ? AND c.event <> ?
    GROUP BY t.key
);
