-- The removals stored before a time.
DELETE FROM file_changes WHERE event = ? AND change_time < ?;
