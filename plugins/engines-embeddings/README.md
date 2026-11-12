# Vector Embeddings

Provides meaning-based search across your vault so you can find related notes, books, and recordings even when they use different words.

<!-- more -->

## What You Can Do

- **Search by meaning, not just keywords**: Find passages that discuss an idea or topic even if they don't contain the exact words you typed.
- **Run privately on your computer or through a connection**: Use the built-in local model offline on your machine, or use a connection to OpenRouter, OpenAI, or another service for fast indexing of large libraries.

## Settings

- **Source**:
  - **Local model**: Runs privately on your computer at no cost and supports over 100 languages. The performance profile of the settings sizes its threads.
  - **Connection**: Sends the texts to a service through a connection you add on the Connections page of the settings. The plugin has no key and no address of its own.
- **Model** (Local model): Pick one of the suggested models from the list: E5 small (the default, 118 MB, runs well on a CPU), Harrier 270M (better results, 344 MB) and BGE-M3 (large, 569 MB). Any other local model needs a profile of your own, described below.
- **Connection** (Connection): One row with a connection and an embedding model. Choosing another connection or model indexes the vault again, and the editor asks first because the service may charge for it. Editing the connection so that its address or the profile of the model changes indexes the vault again as well. If the service rejects the key or asks for payment, indexing stops and the status bar names the connection and the reason.
- **Smallest piece of text to index for meaning**: Pieces of text shorter than this many characters, such as page numbers and running heads, get no vector, so they do not crowd out real answers. They stay searchable by their words. A note shorter than this is found by its words too. The default is 40, and 0 gives every piece a vector. Changing it does not make the other vectors stale: no text is embedded again, and the vectors of the pieces below the new value are removed.
- **Longest chunk of a note, in tokens**: A note is cut into chunks of at most this many tokens, and each chunk gets a vector. The chunk never exceeds the input limit of the model. A smaller value gives more precise passages and more vectors. The default is 256, and the value is held between 16 and 8192. A change cuts the notes again in every open vault; the vectors of texts that did not move are kept.

## Model profiles

A local model is run as its profile says: the profile is data, not code. The profiles that ship with the plugin are in `backend/domain/model/profiles.json`. To add a model or change one without rebuilding, put a `*.json` file in the `profiles` folder of the plugin data directory (the folder the host grants the plugin as `NUMEN_PLUGIN_DATA_DIR`). A file holds one profile or an array of them, and a profile is merged over the shipped profile with the same `id`, field by field, so a file that gives only `queryPrefix` changes that and nothing else. A file that is not valid JSON, names a field the plugin does not know, or holds a profile that cannot run is reported on the log of the plugin and skipped whole. The files are read again each time a setting of the plugin changes.

A profile has these fields. `id` is the Hugging Face repository of a local model, and the value of the Model setting. `title`, `tier` and `sizeMB` describe it in the list. `dims` is the width of the vectors, and `maxTokens` the longest text the model takes; the plugin gives a local model at most 512 tokens whatever this says. `pooling` is how the output becomes a vector: `mean` of the tokens, `cls` (the first token), `last` (the last token), or `output` (the graph answers one vector per text as `sentence_embedding`). `normalize` scales the vector to unit length and is true when absent. `queryPrefix` and `documentPrefix` are put in front of a text that is searched with and a text that is searched, and both may be empty. `onnxFile` is the model file in the repository, such as `onnx/model_quantized.onnx`; a file with a `_data` file beside it is fetched with it. `revision` and `sha256` pin the repository to a commit and each file to its sha256, with the path in the repository as the key, and are optional; a profile that changes `onnxFile` and gives no `sha256` drops the shipped pins. `extraInputs` lists inputs besides the tokens and the mask, and only `position_ids` is known. `floorHint` is the starting similarity a passage has to reach to answer a query, and 0 means unknown. A model with no profile is refused with an error that names the folder, and a model whose graph asks for an input the profile does not list is refused. The variables `NUMEN_EMBEDDING_DIMENSIONS`, `NUMEN_EMBEDDING_MAX_TOKENS`, `NUMEN_EMBEDDING_POOLING`, `NUMEN_EMBEDDING_REPO`, `NUMEN_EMBEDDING_DIR` and `NUMEN_EMBEDDING_FILE` still override the profile that is chosen.

A text is embedded for one of two roles. The index embeds the pieces of your notes as documents, and a question you type is embedded as a query. A model that tells the two apart, such as E5, reads each with the prefix its profile names. A connection is given the prefixes of the profile that its provider reports for the model, which are empty for a model the provider has no profile for.

### After a change of the profile

The vectors are stored under a recipe that holds a short hash of the prefixes, the model file, the token limit and the normalization. For a connection the recipe takes the type of the connection with its address as the source, and the name, width, input limit, pooling and prefixes from the profile that the provider gives before any text is embedded, so the index is fitted before the first batch. The plugin asks for the profile once for each configuration, and the host sends a configuration again when the connection is edited. Any change of those makes the vectors of the old profile stale: the plugin embeds the vault again under the new recipe in the background, and removes the old vectors when it is done. The smallest piece of text to index is not part of the hash. The default model now reads documents and queries with the E5 prefixes, which it did not before, so a vault embedded earlier is embedded once more; this takes about one to two and a half hours for a large vault with the performance profile, and search by words works meanwhile.

## Search Relevance

This plugin tells Numen which model it runs and a starting value for the model's threshold, the similarity a note has to reach to show up in a search. Choosing the threshold, measuring it on your vault and checking a query against it is the job of the **Search Relevance** plugin.
