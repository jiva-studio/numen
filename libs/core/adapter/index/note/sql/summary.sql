SELECT (SELECT COUNT(*) FROM notes WHERE vault_id = ?),
       (SELECT COUNT(*) FROM headings h
          JOIN notes n ON n.source_id = h.note_id
         WHERE n.vault_id = ?);
