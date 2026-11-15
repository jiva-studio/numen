-- Raises the number up to which each feed dropped removals to the newest removal stored before a time.
UPDATE change_feeds SET trimmed_seq = max(trimmed_seq, coalesce((
    SELECT max(seq) FROM file_changes
    WHERE vault_id = change_feeds.vault_id AND event = ? AND change_time < ?
), 0));
