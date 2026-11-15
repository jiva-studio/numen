-- The removals stored longer ago than an age in nanoseconds.
DELETE FROM file_changes WHERE event = ? AND change_time < CAST(unixepoch('subsec') * 1000000000 AS INTEGER) - ?;
