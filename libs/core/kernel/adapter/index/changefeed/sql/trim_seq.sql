-- Raises the number up to which each feed dropped removals to the newest removal stored longer ago
-- than an age in nanoseconds.
UPDATE change_feeds SET trimmed_seq = max(trimmed_seq, coalesce((
    SELECT max(seq) FROM file_changes
    WHERE vault_id = change_feeds.vault_id AND event = ?
      AND change_time < CAST(unixepoch('subsec') * 1000000000 AS INTEGER) - ?
), 0));
