-- Each note type keeps the last sequence of this event that affected it.
INSERT INTO file_changes (vault_id, source_id, event, seq, change_time, path, note_types)
SELECT ?1, ?2, ?3, ?4, CAST(unixepoch('subsec') * 1000000000 AS INTEGER), s.path,
       json_patch(?6, (SELECT json_group_object(type, ?4) FROM
           (SELECT n.type WHERE n.type IS NOT NULL UNION SELECT ?5 WHERE ?5 <> '')))
FROM sources s LEFT JOIN notes n ON n.source_id = s.id
WHERE s.id = ?2 AND s.vault_id = ?1
ON CONFLICT (vault_id, source_id, event) DO UPDATE SET
    seq         = excluded.seq,
    change_time = excluded.change_time,
    path        = excluded.path,
    note_types  = json_patch(file_changes.note_types, excluded.note_types);
