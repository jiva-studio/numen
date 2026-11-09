-- How much disk the index takes: the whole database, and the tables and indexes that hold vectors.
--
-- The size of the vector tables walks their pages, so the caller keeps the answer for a while.
SELECT (SELECT page_count FROM pragma_page_count()) * (SELECT page_size FROM pragma_page_size()),
       coalesce((SELECT sum(pgsize) FROM dbstat WHERE name LIKE 'vectors%' OR name LIKE 'chunks\_vec%' ESCAPE '\'), 0);
