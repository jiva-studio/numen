---
title: Semantic search
description: The model that reads your notes, with running it on your machine, using a service instead, or turning it off.
---

Two of the three bands in [search](/search/) are words, and need nothing. The third finds notes that are _about_ what you typed, and for that your notes have to be read by a model.

That model is provided by the Vector Embeddings plugin. By default it is small, runs on your machine, and is downloaded the first time it is needed. It needs no key and no account, and nothing you write leaves the computer. numen stores the vectors the plugin creates in the vault's index and searches them; it creates none itself.

## Turning it off

Disable the Vector Embeddings plugin. Search is then names and words alone, nothing is downloaded and no network requests are made.

## Using a service

First add a connection on the Connections page of the settings, as described in [Connections](/connections/). Then open the settings of the Vector Embeddings plugin, set **Source** to **Connection**, choose the connection and type the name of an embedding model it offers. Any service that supports the standard embeddings request will work. The plugin holds no key and no address of its own: the connection does, and the settings file holds no key. Set **Source** back to **Local model** to run the model on your computer again.

## Changing the model

The model a vector came from is stored with it, so a vault read by another model is read again. Changing the connection or its model asks first. The question gives the number of notes and the name of the connection, and says that it may cost money. After you confirm, the vault is indexed again with the new model. Search by words works meanwhile, and the vectors of the old model stay on disk, so if you go back to it they are used again. If the service rejects the key or asks for payment, indexing stops and the status bar names the connection and the reason.

## The rest

The setting "Longest chunk of a note, in tokens" (`chunkTokens`) of the Vector Embeddings plugin sets how long a chunk of a note may be. Each chunk has a vector, and a note is cut into chunks of at most that many tokens. It is 256 by default, it is held between 16 and 8192, and it never exceeds the input limit of the model. A smaller value gives more precise passages and more vectors. Changing it cuts the notes of every open vault again; the vectors of the texts that did not move are kept. Without the plugin, notes are cut at a built-in length.

The relevance threshold is how close a passage has to be to a question to count as a result. The Search Relevance plugin measures it on your vault for the model in use and lets you set it by hand. How a model places two pieces of text relative to each other depends on that model, so each model has its own threshold.

The complete list is on [every setting](/reference/).
