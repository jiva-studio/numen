---
title: Semantic search
description: The model that reads your notes — running it on your machine, handing it to a service, or turning it off.
---

Two of the three bands in [search](/finding/) are words, and need nothing. The third finds notes
that are *about* what you typed, and for that your notes have to be read by a model.

Out of the box that model is small, runs on your machine, and is fetched the first time it is
wanted. No key, no account, and nothing you write leaves the computer.

Everything here is `indexing.embedding` in [the settings file](/settings/).

## Turning it off

Search is then names and words alone, nothing is fetched and nothing is asked of a network:

```json
{ "indexing": { "embedding": { "indexing": { "use": "" } } } }
```

## Handing it to a service

Anything speaking the usual embeddings request will do:

```json
{
  "indexing": {
    "embedding": {
      "indexing": {
        "use": "service",
        "service": {
          "base_url": "https://api.openai.com/v1",
          "name": "text-embedding-3-small",
          "key_env": "OPENAI_API_KEY"
        }
      }
    }
  }
}
```

The key is read from the environment variable you name. numen never writes a key back into the
file.

## Two places, because the costs are opposite

`indexing` is where vectors are made while your vault is read. `query` is where the vector for
a question is made. They are separate settings because they are not the same job:

- **Filling the index** is one pass over everything you have. A service does in an hour what a
  laptop does in a day.
- **Asking a question** is twenty words, all day long. Your machine answers in milliseconds
  where a network is a round trip — and answers on a train, with no network at all.

So a good arrangement is often: a service fills the index, your machine answers the questions.
The two must be the same model, and numen checks that they are by embedding the same short text
in both at startup:

```json
{
  "indexing": {
    "embedding": {
      "model": { "name": "bge-m3", "dimensions": 1024, "max_tokens": 512, "pooling": "head" },
      "indexing": {
        "use": "service",
        "service": {
          "base_url": "https://openrouter.ai/api/v1",
          "name": "baai/bge-m3",
          "key_env": "OPENROUTER_API_KEY"
        }
      },
      "query": { "use": "local", "local": { "name": "BAAI/bge-m3", "download": true } }
    }
  }
}
```

The three names differ on purpose: `model.name` is what *you* call the model, and the other two
are what each place calls it.

`query` left unset asks the way the vault was indexed.

## Changing the model

`name`, `dimensions`, `max_tokens` and `pooling` are what a vector is kept under, so changing
any of them means your vault is read again. Nothing is thrown away: setting the old ones back
finds the old work exactly where it was.

`pooling` is the one that goes wrong quietly. A model gathers what a text says either into the
token that opens it or across all of them, and taken the wrong way it answers with vectors that
mean nothing — near nothing, with no error anywhere. `mean` is the E5 family,
`sentence-transformers`, and most of what is published; `head` is BGE, `bge-m3` included.

## The rest

`floor` is how near a question a passage has to stand to count as an answer. Zero takes what the
search was built against. Where a model puts two pieces of text is a fact about that model, so a
model changed is a floor worth measuring again.

`local.dir` names a folder holding a model already on the machine, which is what an installation
with no network uses. `local.file` picks a particular build inside a repository — how a
quantised model is run in place of the full one.

The complete list is on [every setting](/reference/).
