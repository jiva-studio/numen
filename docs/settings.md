# Settings

One file, JSON, named `numen.json` in the folder this desktop keeps a person's configuration in — `~/.config/numen/` on Linux, `~/Library/Application Support/numen/` on a mac, `%AppData%\numen\` on Windows. A run that finds none writes it, holding exactly what that run is doing, so the settings a person changes are the ones in front of them.

Every field left out keeps its default. A file naming one setting is a valid file.

This document is every section of the file, and it is the one place a key is written down. Where what a key does is specified elsewhere, the page that specifies it is linked from the section.

## Appearance

How the window is drawn: how large, which half of a colour pair, and which palette.

```json
{
  "appearance": {
    "interface_scale": 1,
    "text_scale": 1,
    "mode": "system",
    "theme": "preset:numen"
  }
}
```

| | |
| --- | --- |
| `interface_scale` | how large the interface is drawn — its chrome, its controls, the spacing between them and the type in them. 1 is as designed, and it goes from 0.8 to 2. |
| `text_scale` | how large the text a person reads is set: a note, a book, an answer, the editor. 1 is as designed, and it goes from 0.8 to 1.75. |
| `mode` | which half of a colour pair is read: `system`, `light` or `dark`. Any other word is read as `system`. |
| `theme` | the stylesheet the window wears, named by the shelf it came off and its filename: `preset:dracula` ships inside the application, `mine:dracula` is the person's file. |

Each size is a multiplier. `interface_scale` is the root's font size, and every length in the window is a multiple of it: the height of a field and a button, the clearance inside a node, the radii, the spacing, and the type throughout. `text_scale` is a second multiplier over the text a person reads — the editor, and marked-up text with the scale above it, its headings, code, lists and quotations. A hairline, a border, a focus ring and the stroke of a handle are one physical line under either.

A number outside what its setting goes to is refused. What stands in the window's place is in [Starting](starting.md). The number is left as it was written.

A file naming no size at all asks the desktop: a session that set `GDK_DPI_SCALE` draws its interface by that, and one that said nothing is drawn at 1. A scale outside what `interface_scale` goes to is not one it is seeded with.

`-interface-scale` and `-text-scale` say a size for a single launch, over whatever is here. Each stands until that size is chosen in the window, and choosing it is what writes it down.

A file naming `appearance.zoom` is drawn at that number, and the field is given the name `appearance.interface_scale` where it is written. Its value, its place among the fields around it and every other byte of this file stay as they were.

A `zoom` outside what `interface_scale` goes to keeps its name and is said, and the window is drawn as designed until a number in range is written. `zoom: 0` names no size, so the desktop is asked. A file naming a size under both names is drawn at `interface_scale` and keeps both names. A file nothing may be written into is read the same way at every launch, and drawn at the same size.

`zoom` is the reader's word: how large a page of a document is drawn. It is turned in the document in front of a person and it is not a setting.

The tokens a theme sets, where the person's themes live, and how a file names itself light-and-dark or one half only are in [Themes](themes.md).

A theme pinning `color-scheme` is published in one half, and `mode` has nothing left to choose while it is worn. A name matching nothing wears `preset:numen`, and the name that was not found is said; this file is left as it is.

Choosing a theme, light or dark, or either size writes the field it names back here — each is a command of its own in the palette, over the window. The file is read as an object, the named fields are set, and it is written back, so a key this build knows nothing about comes through the write unchanged. A file that does not parse is not written.

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

`model` is what a vector **is**. `indexing` and `query` are where one is **made**.

They are separate because a stored vector outlives the place that made it. One name is run on this machine and served by more than one place, and the numbers each gives for one text are its own — so what a vector is kept under names the model and the address it was made at.

A vault is filled by `indexing` and asked wherever `query` says, and both read the rows `indexing` made. What holds the two together is the comparison between them at startup, and not the name they are called by.

| | |
| --- | --- |
| `model.name` | what the model is called here. Not how either place reaches it: a repository and a service call one model by two names. |
| `model.dimensions` | how wide its vectors are. The coarse index is built for one width, and changing it rebuilds that index from what has been made. |
| `model.max_tokens` | where the model cuts off what it is given. A window cut somewhere else is a window whose vector describes text it does not hold. |
| `model.pooling` | `mean` over the tokens, or `head` from the one that opens the text. |
| `floor` | how near a question a passage stands to be an answer, in cosine similarity. Zero takes what the search was built against. Where a model puts two pieces of text about different things is a fact about that model, so a model changed is a floor measured again. |

Change any of `name`, `dimensions`, `max_tokens` or `pooling` and every stored vector is made again: they are what a vector is kept under. So is where `indexing` makes them — its `service.base_url` and `service.name`, or its `local.name`, `local.dir` and `local.file`. Nothing is thrown away, and setting them back finds the old vectors where they were.

### pooling

A model gathers what a text says either into the token that opens it or across all of them, and taken the wrong way it answers with vectors in a space of its own — near nothing, and no error anywhere.

- `mean` — the E5 family, `sentence-transformers`, most of what is published.
- `head` — BGE, including `bge-m3`.

Where a model's own output is already one vector per text, nothing is pooled and this says nothing about it.

## The two places

Each is `{"use": "local" | "service", "local": {…}, "service": {…}}`. The sections not in use are kept, so the other is a word away.

Trying the other for an afternoon costs nothing under `query`. Under `indexing` it is every vector made again, and the old ones are where they were if it goes back.

`query` left with no `use` asks the way the vault was indexed. Naming it is what separates the two, and the reason to is that their costs are opposite:

- **Filling an index** is a pass over the whole vault, once. A service does in an hour what this machine does in a day.
- **Asking a question** is twenty tokens, all day. This machine answers in milliseconds where a network is a round trip — and answers with no network at all.

Two places are asked whether they are one model: both embed the same short text at startup, and vectors that do not land together mean the second is not used. Nothing in this file could show it — two places name a model by whatever each of them calls it.

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

What comes down is that build and what belongs to it: weights in a second file, a constant in a third, the tokeniser. The other builds in the same folder, and their weights, stay where they are.

A model is fetched and compiled behind the window, and appears in the list of what is being done with the bytes of it that are here. Until it lands a question is answered by the words alone, and the pass that fills the index waits.

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

Anything speaking the `/v1/embeddings` request shape. `base_url` is what points at one.

`batch_characters` bounds one request by everything in it. A count of texts says nothing about their size: the same number of windows carries several times the tokens in transliterated Sanskrit that it does in English.

The key is read from `key` if the file names one, otherwise from the environment variable `key_env` names. It is never written back: rewriting this file is not how a key is set.

## Worked examples, embedding

### Nothing configured

What a first run does: a model on this machine, fetched on first use, no key and no account. The file it writes carries every section filled in — this is the embedding part of it.

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
      },
      "query": { "use": "" }
    }
  }
}
```

### Indexed over a network, asked without one

One model, `bge-m3`, in two places. The service fills the index; the question is embedded here, so search works on a train.

The two names differ because that is what each place calls it — HuggingFace `BAAI/bge-m3`, OpenRouter `baai/bge-m3` — and `model.name` is neither.

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

`multilingual-e5-large` at a quarter of its weight. Same width, so the vector index is not rebuilt — but a quantised model is not the model it was made from, and the vectors are made again.

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

The folder holds the model under the name `file` gives, or `model.onnx`, and `tokenizer.json` beside it.

### No model at all

A vault searched by its words. Nothing is fetched, nothing is asked of a network, and every search is the lexical half answering alone.

```json
{ "indexing": { "embedding": { "indexing": { "use": "" } } } }
```

## Reading a scanned document

`indexing.recognition` is how a scanned page is read. Nothing here runs on its own: a person asks for a reading, and what one is and where it is kept is in [Reading](reading.md).

```json
{
  "indexing": {
    "recognition": {
      "detect": { "expand": 18 }
    }
  }
}
```

| | |
| --- | --- |
| `detect.expand` | how many pixels a found line is widened by, in the image the detector reads: the part of the page scaled to the longest side it is read at. One number serves a heading and a paragraph, because that scaling brings the two to nearly one size. |

The boundary the detector answers with is the text's own outline drawn inside the letters, short by a share of the line's height, and the widening is a flat number of pixels. At 10 the top of every capital and the last letter of every line were cut away. 18 is the middle of where it stops mattering, and it is a setting because it was measured on one book at one resolution — the sweep is in [Performance](performance.md).

## Putting a reading right

`indexing.proofreading` is what corrects a reading. Naming nothing here names no proofreader: a reading is used exactly as it was read, and nothing asks for a key or a network. What a correction may change, and what refuses one, is in [Reading](reading.md).

```json
{
  "indexing": {
    "proofreading": {
      "use": "service",
      "service": {
        "base_url": "https://openrouter.ai/api/v1",
        "batch_url": "https://openrouter.ai/api/beta/batches",
        "name": "google/gemini-2.5-flash",
        "key_env": "NUMEN_PROOFREADING_KEY",
        "pages_at_once": 40,
        "letters_apart": 0.30
      }
    }
  }
}
```

| | |
| --- | --- |
| `use` | `service`, or nothing at all. A section naming no model proofreads nothing. |
| `service.base_url` | anything speaking the `/v1/chat/completions` request shape. |
| `service.batch_url` | the queue pages are left in and collected later, at half the price. Empty asks a page at a time and waits. A batch outlives the run that left it, so one left before the application closed is collected when it opens. |
| `service.name` | which model answers. The name of the model that corrected a line stands beside what it corrected. |
| `service.pages_at_once` | how many pages one request carries. |
| `service.letters_apart` | how far a correction may move a line's letters and still be a correction, as a share of the longer of the two. A correction standing further apart is dropped and that line is left as it was read. |
| `service.key_env` | the environment variable holding the key. |
| `service.key` | the key, where a person writes it into the file. It is never written back: rewriting this file is not how a key is set. |

`letters_apart` is 0.30 because the measured distribution has a hole there. Over 931 corrections of one book, every correction standing further apart than 0.30 was damage — text dragged in from the next line, or one corrected word in place of a whole line — and every one below it was a correction. It is a setting because the next book is not that book; the figures are in [Performance](performance.md).

## Which agent answers

`agent` is which agent answers in the panel, what it may reach, and whether the tools go on a port. What an agent may ask of a vault is in [Agents](agents.md).

```json
{
  "agent": {
    "use": "claude",
    "serve_tools": false,
    "claude": {
      "command": [],
      "model": "",
      "max_steps": 30,
      "reads_hooks_and_skills": false
    }
  }
}
```

| | |
| --- | --- |
| `use` | which agent answers. `claude` is Claude Code, reached by starting it and reading what it prints. Empty answers with none, and the panel says so. |
| `serve_tools` | whether the tools go on a port, which is how an agent a person runs themselves reaches this vault. Off. The agent `use` names is served either way, so an installation naming one has the port open for it. |
| `claude.command` | what starts it: the command line's path, and anything it is started through. Empty asks the path, then the folders its installers write to. Worth naming for an installation those folders do not cover, and for one machine carrying several. |
| `claude.model` | which of its models answers — `opus`, `sonnet`, or a full name. Empty takes whatever that installation answers with. Worth naming because a panel is read while somebody waits. |
| `claude.max_steps` | how many times it may go to the model before it is stopped. 30. |
| `claude.reads_hooks_and_skills` | whether it reads what this machine holds configured for it: hooks, skills, standing instructions in `CLAUDE.md`, plugins. Off. A hook is a shell command Claude Code runs itself, and a question typed into a panel is not asking for one. On, what is configured for this person is read; what a vault carries is refused either way, since a vault arrives from elsewhere. |

A section is kept whether it is the one in use or not, so trying another agent for an afternoon costs nothing.

An installation naming no agent and asking for no tools opens no port and writes no token file: a person who never asked for an agent is running a window and nothing else. Where the port is open, `-mcp-addr` says which address it answers on for a single launch and `-no-mcp` shuts it for one, whatever this file says.

