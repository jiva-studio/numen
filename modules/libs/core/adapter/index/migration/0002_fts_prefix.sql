-- Full-text indexes with prefix indexing for partial-word matching.

DROP TABLE IF EXISTS chunks_fts;
CREATE VIRTUAL TABLE chunks_fts USING fts5 (
    text,
    content='',
    contentless_delete=1,
    prefix='2 3'
);

DROP TABLE IF EXISTS sections_fts;
CREATE VIRTUAL TABLE sections_fts USING fts5 (
    text,
    content='',
    contentless_delete=1,
    prefix='2 3'
);

DROP TABLE IF EXISTS titles_fts;
CREATE VIRTUAL TABLE titles_fts USING fts5 (
    text,
    prefix='2 3'
);
INSERT INTO titles_fts (rowid, text) SELECT source_id, title FROM notes;

DROP TABLE IF EXISTS headings_fts;
CREATE VIRTUAL TABLE headings_fts USING fts5 (
    text,
    prefix='2 3'
);
INSERT INTO headings_fts (rowid, text) SELECT id, text FROM headings;

UPDATE sources SET modified_at = 0;
