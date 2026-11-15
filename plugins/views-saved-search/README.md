# Saved Search

Lets AI assistants in Numen run search queries that combine searches by meaning and by words, narrow them by the type and folder of a file, and leave out what a request excludes.

<!-- more -->

## What It Enables

- **Queries that one search cannot answer**: The assistant joins several phrasings, asks for passages that match two things at once, or leaves out passages about something else, with `meaning("…")`, `words("…")`, `kind` and `path` joined by `&&`, `||` and `!`.
- **Results with their places**: Each result carries the title, the page, chapter or time of its best passage, and a link that opens the source there. Neighboring passages of one file come back as one result.
- **Ordering by relevance**: With a reranking connection, the results are ordered by a reranking model against the request in the user's words; without one, by the similarity of their meaning.

The assistant calls the tool `saved_search_preview`. A query that is not valid comes back with the line and column of each problem.

## Settings

- **Reranking**: Up to five connections that order the results by relevance, tried in order. Add a connection that serves reranking on the Connections page, then choose it here. With no connection, results keep the order of their scores.
- **Default results**: Results a search returns when the assistant names no limit (`20` by default).
- **Most results**: Most results one search returns (`100` by default).
- **Passages per condition**: Passages the index returns for each meaning and words condition (`100` by default).
- **Most passages scored**: Most passages a query is applied to (`1000` by default).
- **Most folders**: Most folders a query joins with `path.startsWith` and `||` (`8` by default).
- **Reranked passages per result**: Passages sent to the reranking service for each result asked for (`3` by default).
- **Most reranked passages**: Most passages sent to the reranking service in one search (`100` by default).
- **Merge gap**: Bytes between two passages of one file that are still joined into one result (`200` by default).
- **Longest query**: Most bytes of the text of one query (`16384` by default).
