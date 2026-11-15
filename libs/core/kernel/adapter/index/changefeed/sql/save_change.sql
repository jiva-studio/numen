-- The latest event of one kind of one source replaces the one before it.
INSERT INTO file_changes (vault_id, source_id, event, seq, change_time) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (vault_id, source_id, event) DO UPDATE SET
    seq         = excluded.seq,
    change_time = excluded.change_time;
