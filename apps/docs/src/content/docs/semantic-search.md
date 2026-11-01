---
title: Semantic search
description: The model that reads your notes, with running it on your machine, using a service instead, or turning it off.
---

Two of the three bands in [search](/search/) are words, and need nothing. The third finds notes that are _about_ what you typed, and for that your notes have to be read by a model.

That model is provided by the Vector Embeddings plugin. By default it is small, runs on your machine, and is downloaded the first time it is needed. It needs no key and no account, and nothing you write leaves the computer. numen stores the vectors the plugin creates in the vault's index and searches them; it creates none itself.

## Turning it off

Disable the Vector Embeddings plugin. Search is then names and words alone, nothing is downloaded and no network requests are made.

## Using a service

In the settings of the Vector Embeddings plugin, set the provider to `openai`, `openrouter` or `custom`, the service URL, and the model name. Any service that supports the standard embeddings request will work. The service's key goes into the same settings, and numen stores it in your system's keychain. The settings file holds no key.

## Changing the model

The model a vector came from is stored with it, so a vault read by another model is read again. Nothing is deleted: if you go back to the old model, its vectors are still there.

## The rest

The key `indexing.embedding.max_tokens` in [the settings file](/settings/) applies to the index, not the model. It is the input limit, in tokens, for a chunk that has a vector; chunks are cut to stay below it. It is 256 by default; a model that accepts more input can use longer chunks.

The relevance threshold is how close a passage has to be to a question to count as a result. The Search Relevance plugin measures it on your vault for the model in use and lets you set it by hand. How a model places two pieces of text relative to each other depends on that model, so each model has its own threshold.

The complete list is on [every setting](/reference/).
