# Vector Embeddings

Powers meaning-based search across your vault so you can find related notes, books, and recordings even when they use different words.

<!-- more -->

## What You Can Do

- **Search by meaning, not just keywords**: Find passages that discuss an idea or topic even if they don't contain the exact words you typed.
- **Run privately on your computer or in the cloud**: Use the built-in local model offline on your machine, or connect to OpenRouter, OpenAI, or another service for fast indexing of large libraries.

## Settings

- **Provider**:
  - **Local**: Runs privately on your computer at no cost and supports over 100 languages.
  - **OpenRouter / OpenAI / Custom service**: Uses an online service to index your library using your **API key**.
- **Model**: Pick one of the suggested models from the list, or type a custom model name (for **Local**, any compatible Hugging Face model name is downloaded automatically on first use).

## Search Relevance

Under **Search relevance** you choose how similar a note has to be to your search before it shows up. Matches below the threshold are left out of the results.

- **Auto / Manual**: In **Auto**, Numen picks the threshold for you. In **Manual**, a slider sets it between 0.05 and 0.95, and a marker on the slider shows the automatic value. A manual threshold applies to the current model only, so each model keeps its own.
- **Threshold and Recalculate**: Shows the threshold in use. **Recalculate** measures it again on your vault.
- **Source**: Says where the threshold comes from: measured on your vault, the starting value of the model, a default, or your settings file. When the settings file sets it, the panel shows the value and you change it in that file.
- **Noise**: The chart shows how similar unrelated notes are to each other, with a line at the threshold. The threshold sits just above this noise, so unrelated notes stay out of your results. Below the chart you see how many titles and passages were compared and when.
- **Not enough notes yet**: Measuring needs at least 20 notes with meaning-based search indexed. Until then Numen uses the starting value of the model.
- **Check a query**: Type a search and press **Check** to see the best match in your vault and whether it passes the threshold.

Changing the provider or the model gives you the threshold that suits the new model. If the embedding service is not running, the panel shows the error and you can try again.
