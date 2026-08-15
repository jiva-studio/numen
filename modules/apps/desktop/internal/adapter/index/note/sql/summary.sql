SELECT (SELECT COUNT(*) FROM notes    WHERE vault_id = ?),
       (SELECT COUNT(*) FROM headings WHERE vault_id = ?);
