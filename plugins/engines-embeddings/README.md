# Vector Embeddings

Provides meaning-based search across your vault so you can find related notes, books, and recordings even when they use different words.

<!-- more -->

## What You Can Do

- **Search by meaning, not just keywords**: Find passages that discuss an idea or topic even if they don't contain the exact words you typed.
- **Run privately on your computer or in the cloud**: Use the built-in local model offline on your machine, or connect to OpenRouter, OpenAI, or another service for fast indexing of large libraries.

## Settings

- **Provider**:
  - **Local**: Runs privately on your computer at no cost and supports over 100 languages.
  - **OpenRouter / OpenAI / Custom service**: Uses an online service to index your library using your **API key**.
- **Model**: Pick one of the suggested models from the list, or type a custom model name (for **Local**, any compatible Hugging Face model name is downloaded automatically on first use).
- **Smallest piece of text to index for meaning**: Pieces of text shorter than this many characters, such as page numbers and running heads, get no vector, so they do not crowd out real answers. They stay searchable by their words. A note shorter than this is found by its words too. The default is 40, and 0 gives every piece a vector. Changing it does not make the other vectors stale: no text is embedded again, and the vectors of the pieces below the new value are removed.

## Search Relevance

This plugin tells Numen which model it runs and a starting value for the model's threshold, the similarity a note has to reach to show up in a search. Choosing the threshold, measuring it on your vault and checking a query against it is the job of the **Search Relevance** plugin.
