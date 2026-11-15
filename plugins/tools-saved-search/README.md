# Saved Search

Keeps search queries that combine searches by meaning and by words as notes of the vault, and shows their results in a tab that stays up to date as the vault changes.

<!-- more -->

## What It Enables

- **Queries that one search cannot answer**: The assistant joins several phrasings, asks for passages that match two things at once, or leaves out passages about something else, with `meaning("…")`, `words("…")`, `kind` and `path` joined by `&&`, `||` and `!`.
- **Saved searches as notes**: A saved search is a note of type `search`. Its body holds the query in a fenced block `numen-query`; its frontmatter holds `rerank`, what makes a result relevant in the user's words, and `limit`, how many results it keeps. Prose outside the block is free.
- **A tab of results**: A note of type `search` opens in the Saved Search tab. It shows the query, the rerank text, when the results were computed, a failed last run with its time and reason, and the results, each with its file, its place (page, time or chapter) and its text. A click, Enter or Space opens the file at that place. **Edit query** opens fields for the query, the rerank text and the limit; **Save** writes them into the note.
- **Results that follow the vault**: When the index changes, a saved search runs again once its last run is at least the refresh delay old; earlier, it is marked stale and runs when the delay ends while the plugin runs, else at the next index change or when its tab opens. A change of its query, rerank text or limit runs it at the next index change, without the delay. A run that fails keeps the last good results.
- **Ordering by relevance**: With a reranking connection, the results are ordered by a reranking model against the rerank text, which reads the whole text of each passage, and each condition gathers more passages for it; without one, by the similarity of their meaning. A run sends the reranking model only the passages whose text it did not score before for the same query, rerank text, floor and connection row.

## Tools

- `saved_search_preview`: runs a query and returns its results without saving anything. A query that is not valid comes back with the line and column of each problem.
- `saved_search_create`: saves a query as a note in the folder the assistant names, else in the folder of the setting, runs it at once and returns the path of the note, its link and how many results the run found.
- `saved_search_update`: changes the query, the rerank text or the limit of a saved search note and runs it again at once.

## The Cache

The plugin keeps the last good results of each saved search in a file of its own folder of the vault store, `.numen/md.numen.tools.saved.search/`, named by the id of the note (`<id>.json`), or by a hash of its path for a note without an id. The file holds what produced the results (hashes of the query, of the rerank text and of the settings, the limit, the floor, the reranking row), the rerank score of each passage, the time of the last good run, the time the index changed after it, and the time and reason of a failed run. Losing the folder costs one run of each search.

## Settings

- **Reranking**: Up to five connections that order the results by relevance, tried in order. Add a connection that serves reranking on the Connections page, then choose it here. With an OpenRouter key, an OpenRouter connection with the model `mongodb/rerank-3` ranks best and answers in a few seconds; without one, the Local reranker runs on this machine and takes tens of seconds for each run. With no connection, results keep the order of their scores.
- **Default results**: Results a search returns when the query names no limit (`20` by default).
- **Most results**: Most results one search returns (`100` by default).
- **Passages per condition**: Passages the index returns for each meaning and words condition when no reranking connection is set (`100` by default).
- **Passages per condition with reranking**: Passages the index returns for each meaning and words condition when a reranking connection is set, with the whole text of each (`300` by default, at most `500`).
- **Most passages scored**: Most passages a query is applied to (`1000` by default).
- **Most folders**: Most folders a query joins with `path.startsWith` and `||` (`8` by default).
- **Passages per result without reranking**: Passages kept for each result asked for when no reranking connection is set (`3` by default).
- **Most passages without reranking**: Most passages kept in one search when no reranking connection is set (`100` by default).
- **Reranked passages per result**: Passages sent to the reranking service for each result asked for (`10` by default).
- **Most reranked passages**: Most passages sent to the reranking service in one search (`300` by default). At this number a cloud reranking service scores the passages of a search in one request.
- **Longest reranked text**: Most bytes of the text of one passage the reranking service reads, from the start of the passage (`4096` by default); the models read about 512 tokens of it.
- **Merge gap**: Bytes between two passages of one file that are still joined into one result (`200` by default).
- **Longest query**: Most bytes of the text of one query (`16384` by default).
- **Refresh delay**: Minutes a saved search waits after its last run before an index change runs it again (`10` by default); `0` runs it at once.
- **Folder of new saved searches**: Folder `saved_search_create` puts a note in when the assistant names none; empty is the vault folder.
