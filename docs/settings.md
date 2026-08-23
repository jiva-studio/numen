# Settings

One file, JSON, at `~/.config/numen/numen.json`. A run that finds none writes
it, holding exactly what that run is doing, so the settings a person changes are
the ones in front of them.

Every field left out keeps its default. A file naming one setting is a valid
file.

This document is the embedding section. The other sections — appearance,
recognition, proofreading, agent — are named here only where they touch it.

## What a vector is, and where it is made

```json
{
  "indexing": {
    "embedding": {
      "model":    { "name": "…", "dimensions": 0, "max_tokens": 0, "pooling": "mean" },
      "indexing": { "use": "local",   "local": {}, "service": {} },
      "query":    { "use": "service", "local": {}, "service": {} },
      "floor": 0.0
    }
  }
}
```

`model` is what a vector **is**. `indexing` and `query` are where one is
**made**.

They are separate because a stored vector outlives the placement that made it.
One model runs on this machine and behind a service, and a vault filled by the
one is asked by the other — so what a vector is kept under names the model and
not the address it came from.

| | |
| --- | --- |
| `model.name` | what the model is called here. Not how either placement reaches it: a repository and a service call one model by two names. |
| `model.dimensions` | how wide its vectors are. The coarse index is built for one width, and changing it rebuilds that index from what has been made. |
| `model.max_tokens` | where the model cuts off what it is given. A window cut somewhere else is a window whose vector describes text it does not hold. |
| `model.pooling` | `mean` over the tokens, or `head` from the one that opens the text. |
| `floor` | how near a question a passage stands to be an answer, in cosine similarity. Zero takes what the search was built against. Where a model puts two pieces of text about different things is a fact about that model, so a model changed is a floor measured again. |

Change any of `name`, `dimensions`, `max_tokens` or `pooling` and every stored
vector is made again: they are what a vector is kept under. Nothing is thrown
away, and setting them back finds the old vectors where they were.

### pooling

A model gathers what a text says either into the token that opens it or across
all of them, and taken the wrong way it answers with vectors in a space of its
own — near nothing, and no error anywhere.

- `mean` — the E5 family, `sentence-transformers`, most of what is published.
- `head` — BGE, including `bge-m3`.

Where a model's own output is already one vector per text, nothing is pooled and
this says nothing about it.

## The two placements

Each is `{"use": "local" | "service", "local": {…}, "service": {…}}`. The
sections not in use are kept, so trying the other for an afternoon costs nothing.

`query` left with no `use` asks the way the vault was indexed. Naming it is what
separates the two, and the reason to is that their costs are opposite:

- **Filling an index** is a pass over the whole vault, once. A service does in
  an hour what this machine does in a day.
- **Asking a question** is twenty tokens, all day. This machine answers in
  milliseconds where a network is a round trip — and answers with no network at
  all.

Two placements are asked whether they are one model: both embed the same short
text at startup, and vectors that do not land in the same place mean the second
is not used. Nothing in this file could show it — two placements name a model by
whatever each of them calls it.

### local

```json
{ "use": "local", "local": {
  "name": "intfloat/multilingual-e5-small",
  "dir": "",
  "file": "",
  "batch_texts": 8,
  "download": true
}}
```

| | |
| --- | --- |
| `name` | a HuggingFace repository. |
| `dir` | a folder holding the model and `tokenizer.json`, used as given. This is what an installation with no network names. |
| `file` | which build inside the repository's `onnx/` folder. Empty is `model.onnx`. Naming another is how a quantised build is run in place of the full one. |
| `batch_texts` | how many texts one forward pass carries. |
| `download` | fetch the model when this machine does not hold it. |

What comes down is that build and what belongs to it: weights in a second file,
a constant in a third, the tokeniser. The other builds in the same folder, and
their weights, stay where they are.

A model is fetched and compiled behind the window, and appears in the list of
what is being done with the bytes of it that are here. Until it lands a question
is answered by the words alone, and the pass that fills the index waits.

### service

```json
{ "use": "service", "service": {
  "base_url": "https://api.openai.com/v1",
  "name": "text-embedding-3-small",
  "batch_characters": 32000,
  "key_env": "NUMEN_EMBEDDING_KEY",
  "key": "sk-…"
}}
```

Anything speaking the `/v1/embeddings` request shape. `base_url` is what points
at one.

`batch_characters` bounds one request by everything in it. A count of texts says
nothing about their size: the same number of windows carries several times the
tokens in transliterated Sanskrit that it does in English.

The key is read from `key` if the file names one, otherwise from the environment
variable `key_env` names. It is never written back: rewriting this file is not
how a key is set.

## Worked examples

### Nothing configured

The file a first run writes. A model on this machine, fetched on first use, no
key and no account.

```json
{
  "indexing": {
    "embedding": {
      "model": {
        "name": "intfloat/multilingual-e5-small",
        "dimensions": 384,
        "max_tokens": 256,
        "pooling": "mean"
      },
      "indexing": {
        "use": "local",
        "local": { "name": "intfloat/multilingual-e5-small", "batch_texts": 8, "download": true }
      }
    }
  }
}
```

### Indexed over a network, asked without one

One model, `bge-m3`, in two places. The service fills the index; the question is
embedded here, so search works on a train.

The two names differ because that is what each place calls it — HuggingFace
`BAAI/bge-m3`, OpenRouter `baai/bge-m3` — and `model.name` is neither.

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
      "query": {
        "use": "local",
        "local": { "name": "BAAI/bge-m3", "download": true }
      }
    }
  }
}
```

### A quantised build, to fit a smaller machine

`multilingual-e5-large` at a quarter of its weight. Same width, so the vector
index is not rebuilt — but a quantised model is not the model it was made from,
and the vectors are made again.

```json
{
  "indexing": {
    "embedding": {
      "model": { "name": "multilingual-e5-large-int8", "dimensions": 1024, "max_tokens": 512, "pooling": "mean" },
      "indexing": {
        "use": "local",
        "local": {
          "name": "intfloat/multilingual-e5-large",
          "file": "model_qint8_avx512_vnni.onnx",
          "download": true
        }
      }
    }
  }
}
```

### A model already on the machine, nothing fetched

```json
{
  "indexing": {
    "embedding": {
      "model": { "name": "bge-m3", "dimensions": 1024, "max_tokens": 512, "pooling": "head" },
      "indexing": {
        "use": "local",
        "local": { "dir": "/opt/models/bge-m3" }
      }
    }
  }
}
```

The folder holds the model under the name `file` gives, or `model.onnx`, and
`tokenizer.json` beside it.

### No model at all

A vault searched by its words. Nothing is fetched, nothing is asked of a
network, and every search is the lexical half answering alone.

```json
{ "indexing": { "embedding": { "indexing": { "use": "" } } } }
```

## The older shape

A file written before questions had a placement of their own names one embedder
at the top, with the model written among its fields:

```json
{ "indexing": { "embedding": {
  "use": "service",
  "service": { "base_url": "…", "name": "baai/bge-m3", "dimensions": 1024, "key": "…" }
}}}
```

It is still read: the placement becomes `indexing`, the model is lifted out of
whichever half `use` names, and questions are asked the way the vault was
indexed.
