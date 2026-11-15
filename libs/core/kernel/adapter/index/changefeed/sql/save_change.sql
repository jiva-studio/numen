-- The latest event of one kind of one source replaces the one before it, stored at the time of now with
-- the path the source has then.
INSERT INTO file_changes (vault_id, source_id, event, seq, change_time, path)
VALUES (?1, ?2, ?3, ?4, CAST(unixepoch('subsec') * 1000000000 AS INTEGER), coalesce((SELECT path FROM sources WHERE id = ?2), ''))
ON CONFLICT (vault_id, source_id, event) DO UPDATE SET
    seq         = excluded.seq,
    change_time = excluded.change_time,
    path        = excluded.path;
