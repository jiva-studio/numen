# Settings

One file, JSON, named `numen.json` in the folder this desktop keeps a person's configuration in — `~/.config/numen/` on Linux, `~/Library/Application Support/numen/` on a mac, `%AppData%\numen\` on Windows. A run that finds none writes it, holding exactly what that run is doing, so the settings a person changes are the ones in front of them.

Every field left out keeps its default. A file naming one setting is a valid file.

A setting the window draws a control for is turned by that control, which patches the file as an object and leaves every other byte of it where it was. The window also opens the file whole, in a tab of its own, from the settings page: what is typed there is written as it stands, and a file the settings cannot be read out of is refused with where in it the trouble is.

This document is every section of the file, and says of each section which keys are worth turning and why. The complete list is the manual's [settings reference](../modules/apps/docs/src/content/docs/reference.md), which is written out of the code itself and checked against it on every build — `indexing.recognition` alone holds twenty-six keys, and three of them are below. Where what a key does is specified elsewhere, the page that specifies it is linked from the section.

## Appearance

How the window is drawn: how large, which half of a colour pair, and which palette.

```json
{
  "appearance": {
    "interface_scale": 1,
    "text_scale": 1,
    "mode": "system",
    "theme": "preset:numen",
    "nest_headings_under_notes": true,
    "nested_heading_limit": 6
  }
}
```

| | |
| --- | --- |
| `interface_scale` | how large the interface is drawn — its chrome, its controls, the spacing between them and the type in them. 1 is as designed, and it goes from 0.8 to 2. |
| `text_scale` | how large the text a person reads is set: a note, a book, an answer, the editor. 1 is as designed, and it goes from 0.8 to 1.75. |
| `mode` | which half of a colour pair is read: `system`, `light` or `dark`. Any other word is read as `system`. |
| `theme` | the stylesheet the window wears, named by its origin and its filename: `preset:dracula` ships inside the application, `mine:dracula` is the person's file. |
| `nest_headings_under_notes` | whether a node nests the headings of the note it stands for under its box. On. |
| `nested_heading_limit` | how many of those headings stand under a node at once, the rest being reached by scrolling. 6, and it goes from 1 to 12. |

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

On, a node in the plex nests the headings of the note it stands for under its box, each at the line it stands on, and choosing one puts that note in front of the person with the keyboard on that line. Off, a node is its box alone, and nothing is asked of the vault about what a note is divided into.

`Nest headings under notes` is a command of its own in the palette, over the window. Choosing `On` or `Off` writes this field back here, and every plex the window holds is drawn again from what was written.

## A day of review

`review` is what a day of review is, on this person's clock. How a deck is scheduled is not here: it is in the vault, in the preset the deck points at — [Cards](cards.md).

```json
{
  "review": {
    "day_start_time": "04:00"
  }
}
```

| | |
| --- | --- |
| `day_start_time` | the hour a day of review begins at, on the clock on the wall. `04:00`, and it goes from `00:00` to `12:00`. |

An answer given before that hour is written into the day before: a person answering at one in the morning is finishing the evening they sat down in, and a boundary at midnight would cut one session in two. The hour is an hour on the wall, so a day is read where a person reads it — [Flashcards](flashcards.md).

Anything that is not an hour of the day is said, and `04:00` stands. The file is left as the person wrote it.

## What a note is called

`naming` is how a note's title and the name of its file are held together.

```json
{
  "naming": {
    "sync_title_and_filename": true
  }
}
```

| | |
| --- | --- |
| `sync_title_and_filename` | whether renaming either of the two brings the other into line. On. |

A note is shown by its `title`, else by its filename — [Note format](note-format.md). On, giving a note a different name renames its file after that name, and renaming its file writes the new name into the key where the note carries one. Off, the two are told apart in both directions: a new name is written into the note and the file stays where it is, and a renamed file leaves the note as it was written.

A note carrying no `title` is named by its file, and nothing else in it can carry a name. Renaming such a note renames its file whichever way this is set, and no key is written into it unless the filename cannot carry the whole title.

Renaming a file into another folder is not renaming a note, and neither is renaming a folder. Neither changes what a note is called.

Typing a new `# Heading` into a note does not rename its file. A save puts down the text a person typed and adds nothing to it.

`Sync title and filename` is a command of its own in the palette, over the window. Choosing `On` or `Off` writes this field back here, and the next rename reads what was written.

## Performance profiles

`performance` governs how intensively background AI pipelines (vector embedding, OCR, transcription, and agent concurrency) utilize system hardware.

```json
{
  "performance": {
    "profile": "balanced",
    "custom": {
      "embedding_batch_size": 32,
      "ocr_threads": 4,
      "ocr_sessions": 1,
      "llm_concurrency": 2
    }
  }
}
```

| | |
| --- | --- |
| `profile` | the workload profile: `eco`, `balanced` (default), `maximum`, or `custom`. |
| `custom.embedding_batch_size` | how many passages are embedded in a single forward pass when `profile` is `custom`. |
| `custom.ocr_threads` | intra-op thread count per OCR model instance. |
| `custom.ocr_sessions` | concurrent OCR inference workers. |
| `custom.llm_concurrency` | maximum simultaneous LLM/agent requests in flight. |

### Hardware adaptation

When `profile` is `eco`, `balanced`, or `maximum`, numen reads the machine and resolves every pool from it. The machine is described by its logical CPUs (hardware threads the process may run on, limited by the process affinity and, on Linux, the cgroup v2 CPU quota), its physical cores, its performance cores (the cores of the highest performance class; equal to the physical cores when none differ), and its installed and free memory. A figure that cannot be read is unknown and takes the fallback below.

The **core budget** is the number of hardware threads CPU-bound work may occupy at once: a share of the hardware threads of the performance cores. A performance core is counted with as many threads as the machine has logical CPUs per physical core, rounded up. When the core counts are unknown, every logical CPU counts as a core.

| | Eco | Balanced *(default)* | Maximum |
| --- | --- | --- | --- |
| Core budget | 1 | 50% of the performance threads, at least 1 | 100% of the performance threads |
| OCR sessions × threads | 1 × 1 | the budget, split as below | the budget, split as below |
| Workers (general CPU-bound pool) | 1 | half the budget, at least 1 | the budget |
| Embedding batch | 16 | 1 per 256 MB free, from 16 to 64 | 1 per 128 MB free, from 32 to 256 |
| LLM concurrency | 1 | 2 | 4 |
| Memory limit (soft) | 25% of installed memory | 50% of installed memory | 75% of installed memory |

**Where each figure is read.** The resolved plan is the source of every limit below; a limit the user sets in a section of its own wins over it, and a component built without a plan keeps the fallback in the last column.

| Consumer | Limit | Fallback without a plan |
| --- | --- | --- |
| Local embedding batch | embedding batch, unless the model names `batch_texts` | the model's own |
| Local embedding, threads of the loaded model | core budget, unless the model names `threads` | the default of the runtime |
| Local embedding, forward passes at once | workers, lowered to what a quarter of the available memory holds, unless the model names `sessions`; one on an accelerator | one up to 2 CPUs, two up to 4, three beyond |
| Search question and indexing sharing one local model | indexing holds one pass fewer than the model runs at once, at least one; a question is never held | not applicable |
| OCR | OCR threads and sessions | unresolved counts, then named defaults |
| Proofreading, batches one call answers | LLM concurrency, unless the profile names `in_flight` | 4 |
| Embedding service, requests in flight | LLM concurrency | 4 |
| Search, sources extracted at once | workers | 4 |
| Cards, decks parsed and curve places worked out at once | workers | one per CPU |
| PDF worker pool | workers, at least 2; the workers kept alive between documents are the planned workers, lowered to what 5% of the available memory holds at 32 MB each, none when it is unknown, and all are released after 5 minutes without an open document | one per CPU, at least 2; none kept alive |
| Transcription, resampling and feature extraction | workers | one per CPU |

OCR threads × sessions never exceeds the core budget. The budget is split into 4 sessions, or more when one session would need over 8 threads; sessions are capped at 16 and at the budget, and each session gets the budget divided by the sessions, at most 8 threads.

The memory limit is a soft limit on the memory the Go runtime manages, set once at startup from the installed memory: near it the collector runs more often, and no allocation fails. It is not set when the installed memory is unknown, when `GOMEMLIMIT` is set, or when a limit is already in force. Custom follows the Balanced share.

Free memory is what the operating system reports as available. When only installed memory is known, half of it is taken as free; when neither is known, 4096 MB is taken as free. macOS does not report free memory, so it takes the second rule.

- **Eco**: Keeps background work unobtrusive and battery-friendly: one thread for each pool.
- **Balanced**: Optimized for everyday responsive use; half the performance threads, so the interface keeps the rest.
- **Maximum**: Full use of the performance cores for fast initial indexing and bulk processing.
- **Custom**: Uses the explicit parameters written in `performance.custom`, clamped to their bounds: batch size up to 512 (32 when unset), OCR threads up to 64, OCR sessions up to 16, LLM concurrency from 1 to 16. When both `ocr_threads` and `ocr_sessions` are set they are used as written, even if their product exceeds the core budget. When one is unset, it is derived so that the product fills the Maximum budget; when both are unset, the Maximum split applies. Workers follow the Balanced rule.

`Set performance profile` is a command of its own in the palette, over the window. Choosing a profile updates the configuration immediately.

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
| `model.dimensions` | how wide its vectors are. The coarse index is built for one width and records the recipe that filled it, and changing the width or the model rebuilds it from the vectors already kept, without a model call. |
| `model.max_tokens` | where the model cuts off what it is given. A window cut somewhere else is a window whose vector describes text it does not hold. |
| `model.pooling` | `mean` over the tokens, or `head` from the one that opens the text. |
| `floor` | how near a question a passage stands to be an answer, in cosine similarity. Zero takes what the search was built against. Where a model puts two pieces of text about different things is a fact about that model, so a model changed is a floor measured again. |

Change any of `name`, `dimensions`, `max_tokens` or `pooling` and every stored vector is made again: they are what a vector is kept under. So is where `indexing` makes them — its `service.base_url` and `service.name`, or its `local.name`, `local.dir` and `local.file`. Nothing is thrown away, and setting them back finds the old vectors where they were.

### pooling

A model gathers what a text says either into the token that opens it or across all of them, and taken the wrong way it answers with vectors in a space of its own — near nothing, and no error anywhere.

- `mean` — the E5 family, `sentence-transformers`, most of what is published.
- `head` — BGE, including `bge-m3`.

Where a model's own output is already one vector per text, nothing is pooled and this says nothing about it.

## The two providers

Each is `{"use": "local" | "service", "local": {…}, "service": {…}}`. The sections not in use are kept, so the other is a word away.

Trying the other for an afternoon costs nothing under `query`. Under `indexing` it is every vector made again, and the old ones are where they were if it goes back.

`query` left with no `use` asks the way the vault was indexed. Naming it is what separates the two, and the reason to is that their costs are opposite:

- **Filling an index** is a pass over the whole vault, once. A service does in an hour what this machine does in a day.
- **Asking a question** is twenty tokens, all day. This machine answers in milliseconds where a network is a round trip — and answers with no network at all.

Two providers are asked whether they are one model: both embed the same short text at startup, and vectors that do not land together mean the second is not used. Nothing in this file could show it — two providers name a model by whatever each of them calls it.

### local

```json
{ "use": "local", "local": {
  "name": "intfloat/multilingual-e5-small",
  "dir": "",
  "file": "",
  "batch_texts": 8,
  "engine": "",
  "runtime": "",
  "threads": 4,
  "download": true
}}
```

| | |
| --- | --- |
| `name` | a HuggingFace repository. |
| `dir` | a folder holding the model and `tokenizer.json`, used as given. This is what an installation with no network names. |
| `file` | which build inside the repository's `onnx/` folder. Empty is `model.onnx`. Naming another is how a quantised build is run in place of the full one. |
| `batch_size` | how many texts one forward pass carries (`batch_texts` is supported as an alias). |
| `engine` | what runs the model: `onnxruntime`, the library the host provisions, or `go`, the backend in the binary. Empty takes the runtime where this platform has one published and the Go backend where it has none. The vectors are the same either way, so the index does not record which ran. |
| `runtime` | the ONNX Runtime shared library, when the engine is `onnxruntime`. Empty takes the library the host provisioned for the plugin. |
| `threads` | how many threads the loaded model runs its passes on. Left out, it is the performance profile's core budget. |
| `sessions` | how many forward passes run at once on the one loaded model. Left out, it is the performance profile's workers. A model running on an accelerator runs one pass unless this is set. |
| `download` | fetch the model when this machine does not hold it. |

What comes down is that build and what belongs to it: weights in a second file, a constant in a third, the tokeniser. The other builds in the same folder, and their weights, stay where they are.

A model is fetched and compiled behind the window, and appears in the list of what is being done with the bytes of it that are here. Until it lands a question is answered by the words alone, and the pass that fills the index waits.

### service

```json
{ "use": "service", "service": {
  "base_url": "https://api.openai.com/v1",
  "name": "text-embedding-3-small",
  "batch_characters": 32000
}}
```

Anything speaking the `/v1/embeddings` request shape. `base_url` is what points at one.

`batch_characters` bounds one request by everything in it. A count of texts says nothing about their size: the same number of windows carries several times the tokens in transliterated Sanskrit that it does in English.

The service's key is entered in the app, in the settings of the Vector Embeddings plugin, and kept in the system keychain. This file holds no key, and no environment variable is read for one. An older file that names `key` or `key_env` here loads, and both are ignored.

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
          "name": "baai/bge-m3"
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

## Recognising a scanned document

`indexing.recognition` is how a scanned page is read. Nothing here runs on its own: a person asks for a recognition, and what one is and where it is kept is in [Recognising a document](recognising.md).

```json
{
  "indexing": {
    "recognition": {
      "detect": { "expand": 18 },
      "proofread": { "with": "", "automatically": false }
    }
  }
}
```

| | |
| --- | --- |
| `detect.expand` | how many pixels a found line is widened by, in the image the detector reads: the part of the page scaled to the longest side it is read at. One number serves a heading and a paragraph, because that scaling brings the two to nearly one size. |
| `proofread.with` | which profile under `indexing.proofreading.profiles` puts a recognition right. Empty proofreads nothing, and a recognition is used exactly as it was read. A name no profile carries is an error at startup. |
| `proofread.automatically` | whether a recognition is proofread as soon as it is finished. Off leaves it to the hand: a person asks for it on the book in front of them. |

The boundary the detector answers with is the text's own outline drawn inside the letters, short by a share of the line's height, and the widening is a flat number of pixels. At 10 the top of every capital and the last letter of every line were cut away. 18 is the middle of where it stops mattering, and it is a setting because it was measured on one book at one resolution — the sweep is in [Performance](performance.md).

## Listening to a recording

`indexing.transcription` is how a recording is transcribed. A recording carries no text of its own, so its transcript is the only text there is: a recording the vault holds none for is transcribed without anybody asking, and `indexing.transcribe_recordings` is what stops that. What a transcript is and where it is kept is in [Transcribing](transcribing.md).

```json
{
  "indexing": {
    "transcribe_recordings": true,
    "transcribe_max_mb": 300,
    "transcription": {
      "download": true,
      "threads": 4,
      "model": { "name": "parakeet-tdt-0.6b-v3-int8" },
      "segmenter": { "name": "silero-vad", "threshold": 0.5, "silence": 500, "pad": 200, "longest": 30000, "shortest": 100, "least": 2500 },
      "proofread": { "with": "", "automatically": false }
    }
  }
}
```

| | |
| --- | --- |
| `transcribe_recordings` | whether a recording the vault holds no transcript for is transcribed on its own. On. A vault of a hundred hours is a day of a machine, and turning this off leaves it to the hand — the command line's `transcribe`, and the tool an agent asks through. |
| `transcribe_max_mb` | how large a recording may be and still be transcribed unasked, in megabytes. 300, which is a talk of a few hours. A larger one waits to be asked for by name, because a folder of albums is days of a machine. A negative number is no limit. |
| `runtime` | the ONNX Runtime shared library. Empty takes the library the host provisioned for the plugin, downloaded at the release pinned by sha256 into the models folder. One process opens one, and both a recognition and a transcription run their models through it. |
| `dir` | a folder holding the models. Empty takes the folder beside the application, and then the download cache. |
| `download` | whether what is not on this machine may be fetched. A model is fetched from the address named here and checked against no sum: the runtime's address is this build's and carries one, and a model's is the person's own setting, which nobody but them could publish a sum for. A model named here is a model trusted. |
| `threads` | how many threads one model may use. 4. |
| `model.name` | what the transducer is called in the record kept beside a transcript. |
| `model.from` | the folder its four files are fetched from. The encoder, the decoder, the joiner and the tokens are one model: three graphs from two exports write nothing anybody can read. |
| `model.encoder`, `model.decoder`, `model.joiner`, `model.tokens` | the files on this machine. A path is used as given; an empty one is the file of that name under `model.from`. |
| `segmenter.name` | what the segmenter is called in that same record. Where a segment is cut is part of what the words are, so it is named beside the model that transcribed them. |
| `segmenter.from`, `segmenter.path` | where the segmenter is fetched from, and a file on this machine. A path is used as given; `from` is looked for in `dir` first. |
| `segmenter.threshold` | how sure the model has to be that a window carries speech. 0.5. |
| `segmenter.silence` | how much quiet, in milliseconds, closes a segment. 500. |
| `segmenter.pad` | how many milliseconds are kept on each side of a segment. 200, because the model answers on the window a sound begins in, and the sound before that window is what the first letter of the word is made of. |
| `segmenter.longest` | how many milliseconds one segment may run to. 30000. One segment is one run of the encoder, and its cost grows with its length; speech going on longer is cut at the quietest window this side of the limit. |
| `segmenter.shortest` | how many milliseconds a segment carries to be a segment at all. 100. |
| `segmenter.least` | how many milliseconds a segment runs to before it stands as a line of its own. 2500. A shorter one is put together with the segment after it, up to `longest`. A line of a transcript is read, so it holds a phrase; and the model transcribes a sentence better than it transcribes a word out of one. |
| `proofread.with` | which profile under `indexing.proofreading.profiles` puts a transcript right. Empty proofreads nothing, and a transcript is used exactly as it was transcribed. A name no profile carries is an error at startup. |
| `proofread.automatically` | whether a transcript is proofread as soon as it is finished. Off leaves it to the hand: a person asks for it on the recording in front of them. |

`transcribe_recordings` and `transcription.proofread.automatically` are two flags about two things. The first decides whether a recording nobody asked about is listened to at all. The second decides whether a transcript that already exists is put right by itself. An installation can hear every recording unasked and proofread none of them, and it can proofread every transcript it has while listening to nothing new.

Recognising a scan and transcribing a recording each hold the models and the processor, so they take turns: a person who asked for a scan to be read waits for it before a recording is transcribed, and the one waiting says so in the list of what is being done.

Every recording handed over ends in an answer, and only one of them is "later". Words are an answer, a recording carrying no speech is an answer, and a file nothing here can open is an answer; all three are written down and the recording is not listened to again. Bytes another run holds are the one ending that means come back later. Asking for a recording to be tried again is taking its answer away.

## Putting a text right

`indexing.proofreading` is what corrects a text a model produced — a recognition of a scan, a transcript of a recording. It stands beside `recognition` and `transcription` because a text put right is a text searched, and it holds the profiles both of them name. What a correction may change, and what refuses one, is in [Proofreading](proofreading.md).

An installation naming no profile proofreads nothing, and nothing asks for a key or a network.

A profile says how a text is cut into requests. Which model answers them is not a profile's to say: the proofreading plugin asks the host with the purpose `proofreading`, and the route of that purpose under [`ai.routes`](#where-a-plugins-questions-to-a-model-are-answered) names the agent plugin, the model and the number of questions answered at once. Without a route, proofreading fails with "unavailable" and nothing is corrected.

```json
{
  "indexing": {
    "proofreading": {
      "max_edit_distance": 0.30,
      "profiles": {
        "scans": { "batch_size": 40, "overlap": 0 },
        "speech": { "batch_size": 60, "overlap": 2, "in_flight": 3 }
      }
    }
  }
}
```

| | |
| --- | --- |
| `max_edit_distance` | how far a correction may move a line's letters and still be a correction: the Levenshtein distance between what is left after spaces, punctuation, symbols, diacritics and case come off, as a share of the longer of the two. Left out, the proofreading plugin's own limit, 0.30. A correction standing further apart is dropped and that line is left as it was. It stands above the profiles because it is one threshold for the installation: how far a correction may move says nothing about what it was asked for through. |
| `profiles` | a map of name to profile. The name is what a consumer says under `proofread.with`, and it is the person's own word. |

A profile is flat: every key sits at the profile's own level.

| | |
| --- | --- |
| `batch_size` | how many lines one speech request carries, or how many pages one scan call carries. 40. |
| `overlap` | how many lines neighbouring batches share, so a phrase torn at a batch boundary is still seen whole by one of them. A line two batches both answered about is taken from the later of the two, which is the batch that saw more of what follows it. |
| `in_flight` | how many batches one call answers. Left out, it is the performance profile's LLM concurrency; with no performance profile, 4. How many questions reach the model at once is the concurrency of the route. |

A profile of an older file that names `use`, `name`, `base_url`, `batch_url`, `key`, `key_env`, `model`, `command`, `attempts`, `delay_ms` or `max_delay_ms` is read for its sizes alone. The command line a person has installed is one agent plugin behind a route, and a key belongs to the agent that holds it.

`max_edit_distance` is 0.30 because the measured distribution has a hole there. Over 931 corrections of one book, every correction standing further apart than 0.30 was damage — text dragged in from the next line, or one corrected word in place of a whole line — and every one below it was a correction. It is a setting because the next book is not that book; the figures are in [Performance](performance.md).

## Worked example, proofreading

Scans and transcripts through the `claude` command line on this machine, two questions at once. Both run on their own: a book read is a book proofread, and a recording transcribed is a recording proofread, with nobody asked.

```json
{
  "ai": {
    "routes": {
      "proofreading": { "agent": "md.numen.agent-claude", "model": "haiku", "concurrency": 2 }
    }
  },
  "indexing": {
    "proofreading": {
      "max_edit_distance": 0.30,
      "profiles": {
        "scans": { "batch_size": 40, "overlap": 0 },
        "speech": { "batch_size": 60, "overlap": 2, "in_flight": 3 }
      }
    },

    "recognition":   { "proofread": { "with": "scans",  "automatically": true } },
    "transcription": { "proofread": { "with": "speech", "automatically": true } }
  }
}
```

A scan goes forty pages to a call. A page ends where a page ends, so nothing carries over and `overlap` is 0. Speech goes sixty cues to a request, three requests to a call; a sentence runs across the cue a batch ends on, so two cues are shared with the batch on either side.

A `with` naming a profile `profiles` does not carry is an error at startup. An installation that meant to proofread and misspelled the name is told so, and does not run quietly proofreading nothing.

## Reaching an address

`importing` is how what a url points at is fetched, and where the tools that fetch it are. What is kept, and where, is [Importing an address](importing.md).

```json
{
  "importing": {
    "captions": ["en"],
    "automatic_captions": true,
    "copy_max_size_mb": 500,
    "copies_to_vault": false,
    "yt_dlp": { "command": [], "arguments": [] },
    "ffmpeg": { "command": [], "arguments": [] }
  }
}
```

| | |
| --- | --- |
| `captions` | which languages published words are preferred in, best first. Empty takes the language the video was spoken in. |
| `automatic_captions` | whether words a machine wrote count where a person published none. On. |
| `copy_max_size_mb` | how large a copy of a video may be. A copy that runs past it stops there and nothing is kept. |
| `copies_to_vault` | whether a copy is kept beside the url, under that file's own name, as a file of the person's own. Off, which keeps it in the application's folder, where losing it costs another fetch. |
| `yt_dlp.command`, `ffmpeg.command` | what starts it: the tool's path, and anything it is started through. Empty asks the `PATH`. |
| `yt_dlp.arguments`, `ffmpeg.arguments` | what every run is handed before its own arguments. |

**A command is a list, and that is the whole answer to a machine with no fixed paths.** A tool that lives where nothing may write a path down is started through whatever does know — `["nix", "run", "nixpkgs#yt-dlp", "--"]`, a wrapper on the `PATH`, a store path in full. Nothing here has an opinion about how a machine keeps its tools.

`arguments` is what answers a site that refuses an unattended request. Cookies from a browser, an extractor argument carrying a token, a runtime that mints one: each is that machine's own, is passed through as it stands, and what the tool said when it refused is what the person is shown.

None of this is turned in the window. Every key here is a machine's answer rather than a person's taste, and a control for it would be a control for something set once, on the day the machine was set up.

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
| `claude.model` | which of its models answers — `opus`, `sonnet`, `haiku`, or a full name. Empty takes whatever that installation answers with. Worth naming because a panel is read while somebody waits. |
| `claude.max_steps` | how many times it may go to the model before it is stopped. 30. |
| `claude.reads_hooks_and_skills` | whether it reads what this machine holds configured for it: hooks, skills, standing instructions in `CLAUDE.md`, plugins. Off. A hook is a shell command Claude Code runs itself, and a question typed into a panel is not asking for one. On, what is configured for this person is read; what a vault carries is refused either way, since a vault arrives from elsewhere. |

A section is kept whether it is the one in use or not, so trying another agent for an afternoon costs nothing.

An installation naming no agent and asking for no tools opens no port and writes no token file: a person who never asked for an agent is running a window and nothing else. Where the port is open, `-mcp-addr` says which address it answers on for a single launch and `-no-mcp` shuts it for one, whatever this file says.


## Where a plugin's questions to a model are answered

`ai.routes` is where a plugin that holds the `ai:complete` permission has its questions answered. A plugin names a purpose (`proofreading`) and never a model, an agent or a key; the route of that purpose says which agent plugin answers, with which model, and how many questions it answers at once. A purpose without a route of its own is answered by the route named `default`. With neither, the plugin is told there is no route, and nothing is asked of any model.

```json
{
  "ai": {
    "routes": {
      "default": { "agent": "", "model": "", "concurrency": 1 },
      "proofreading": { "agent": "md.numen.agent-claude", "model": "sonnet", "concurrency": 2 }
    }
  }
}
```

| | |
| --- | --- |
| `agent` | the id of the agent plugin that answers. Empty is the plugin the installation provides for the `ai.completion.provider` capability. A plugin that is not installed, or does not declare the capability, leaves the plugin told that the agent is unavailable. |
| `model` | the model the agent answers with. Empty leaves the choice to the agent. |
| `concurrency` | how many questions the route answers at once; the others wait for a place, and a question whose plugin gave up leaves the wait. Below one is one. |

The file is read at every question, so a route written while the application runs applies to the next one.

No key is written in this file. An agent plugin that needs one declares it under `secrets` in its manifest; the host keeps it in the system keychain and hands it to the plugin in `Configure`, so it is in neither the plugin's environment nor its command line. Under `plugins.<id>.storedSecrets` the host records only the names of the secrets that have a key, and reads the keychain for those alone. Storing a key on a machine without a keychain fails with an error saying so and keeps the key nowhere else; the plugin is then configured without it.
