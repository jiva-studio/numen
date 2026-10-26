---
title: Semantic search
description: The model that reads your notes — running it on your machine, handing it to a service, or turning it off.
---

Two of the three bands in [search](/search/) are words, and need nothing. The third finds notes that are *about* what you typed, and for that your notes have to be read by a model.

That model belongs to the Vector Embeddings plugin. Out of the box it is small, runs on your machine, and is fetched the first time it is wanted. No key, no account, and nothing you write leaves the computer. numen keeps the vectors the plugin makes in the vault's index and searches them; it makes none itself.

## Turning it off

Disable the Vector Embeddings plugin. Search is then names and words alone, nothing is fetched and nothing is asked of a network.

## Handing it to a service

In the settings of the Vector Embeddings plugin, set the provider to `openai`, `openrouter` or `custom`, the service URL, and the model name. Anything speaking the usual embeddings request will do. The service's key goes into the same settings, and numen keeps it in your system's keychain. The settings file holds no key.

## Changing the model

The model a vector came from is kept beside it, so a vault read by another model is read again. Nothing is thrown away: going back to the old model finds the old vectors exactly where they were.

## The rest

Two keys under `indexing.embedding` in [the settings file](/settings/) are about the index, not the model:

- `max_tokens` is the input bound, in tokens, a chunk that carries a vector is cut below. It is 256 by default; a model that reads more can take longer chunks.
- `floor` is how near a question a passage has to stand to count as an answer. Zero takes what the search was built against. Where a model puts two pieces of text is a fact about that model, so a model changed is a floor worth measuring again.

The complete list is on [every setting](/reference/).
