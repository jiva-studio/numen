-- The index of headings is addressed by the heading's own number, which is its
-- rowid. A heading carries no vault of its own, and belongs to the vault its
-- note does.
DELETE FROM headings_fts WHERE rowid IN (
    SELECT headings.id
    FROM headings
    JOIN notes ON notes.source_id = headings.note_id
    WHERE notes.vault_id = ?
);
