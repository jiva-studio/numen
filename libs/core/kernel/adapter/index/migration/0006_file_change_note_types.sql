ALTER TABLE file_changes ADD COLUMN note_types TEXT NOT NULL DEFAULT '{}';

UPDATE file_changes AS c
SET note_types = json_object(n.type, c.seq)
FROM notes AS n
WHERE c.source_id = n.source_id AND c.vault_id = n.vault_id AND c.event <> 'file.removed';

-- A cursor that missed a removal without a recorded note type requires a full read.
UPDATE change_feeds AS f
SET trimmed_seq = max(f.trimmed_seq, coalesce((
    SELECT max(c.seq) FROM file_changes c
    WHERE c.vault_id = f.vault_id AND c.event = 'file.removed'
), 0));
