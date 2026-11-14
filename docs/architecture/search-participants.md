# Search participants

A search in the core asks every participant the same query and merges their rankings by one method. The core owns the index and all of its storage; a participant never opens it. The words of the index and the names of its sections are participants built into the core. A plugin with the capability `search.source` adds one more through `numen.v1.SearchSourceService`; Vector Embeddings is that plugin, and it keeps its vectors in the place the core gives it inside the core index database.

## What the core learns from a source

`DescribeSearchSource` answers, and the core reads it again each time the settings of the plugin change:

- `name`: the text a person knows the data by, shown in statistics and the log.
- `recipe`: an opaque string. Data kept under another recipe is stale. The core appends how it stores a vector (int8 at its scale) and keys every stored vector and every stored floor by the result.
- `width`: how many values one vector of the source holds. The core sizes its vector table by it.
- `chunk_tokens`: the most tokens one chunk holds; the core cuts notes and file texts at it, and at its own default where it is zero.
- `min_text_length`: the shortest chunk text the source is handed; a shorter chunk is searched by the built-in participants only.
- `concurrency` and `floor_hint`: how many calls of `IndexTexts` run at once, and the floor a passage starts out having to reach.

The core knows no model: no pooling, no handling, no provider, no token limit.

## Indexing

The core decides what is indexed and when (vault changes, text versions, cut recipes). For each chunk with no vector under the recipe it reads the text and calls `IndexTexts(place, texts)`, in batches, `concurrency` at a time. A text carries the chunk id, the path of its source and the id of the text version it was cut from (empty for a note). The place names the vault and the recipe. The source answers one vector per text, in order, and the core quantizes them and writes them into the place. A text whose content hash already has a vector under the recipe is not handed again, and equal texts of one batch are handed once.

## Query

`SearchChunks(place, query)` gets the query text, the kinds, the path prefix, the limit and the floor the query names (zero: the floor of the vault). The source answers ranked passages: a chunk id the core handed it and a score, best first. Vector Embeddings embeds the query (it keeps the vectors of recent queries by recipe) and asks the host for the nearest chunks of its place with `HostSearchService.FindNearestChunks(place, vector, ...)`; the host applies the floor of the vault under the recipe (stored, then the hint, then the default). The core turns the chunk ids into the passages that enclose them (note, range, line), in the order of the source.

## Merge

Every participant implements `Rank(vault, query) Ranking`: passages best first. The core fuses all rankings with reciprocal rank fusion (k = 60) and keeps at most `Each` passages of one source. A ranking marked leading (the section names) puts its first passage of a source ahead of the other passages of that source. The merge does not know which participant is built in. A source that fails is reported and left out; the other rankings answer. The parameters of a search name how many candidates the words, the names and the sources keep, and the window's modes set one of the three.

## Ready event

After an indexing pass leaves no chunk of a vault pending for the source, the core emits `onSearchSourceReady:<vault id>`. A plugin subscribes with the activation event `onSearchSourceReady`; it replaces `onVectorsReady`.

## Calibration

`HostSearchService` keeps `GetEmbeddingState`, `GetIndexStats`, `EmbedTexts`, `SampleSimilarities`, `GetBestSimilarity` and the floor calls for tools-search-relevance and tools-index-stats. `EmbedTexts` is answered by the source's `EmbedQueries`.

## Limits

One source at a time: the vector table has one width, so the core opens the first plugin that declares `search.source`. The search and the merge take any number of participants. A read of the description the settings do not allow (a connection source with no connection) fails with the sentence the plugin gives, which the window lists as the failure of the search source. A window opens such a source all the same, with no description: it takes no part in search or indexing. Each time the plugin is given its configuration (a setting of its own, an edit of a connection it uses, a start of its process) the core reads the description again; once the read succeeds the source joins search, its texts are indexed and `onSearchSourceReady` fires, with no reopen of the window. A command of the command line reads the description once, and runs on words alone when the read fails.
