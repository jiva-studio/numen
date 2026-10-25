---
title: Settings reference
description: Every key numen.json can hold, section by section, taken from the application itself.
---

Every key `numen.json` can hold. The pages under Settings say which of these are worth touching and why; this one is here so that nothing is missing.

Each heading below is a section of the file, and the keys under it are written the way they are written inside that section — `model.name` under `indexing.embedding` is `indexing.embedding.model.name` if you write it out in full.

A key you leave out keeps its default, so a file naming one setting is a complete file. A key numen does not know is carried through untouched. Under `agent`, each program an agent plugin lists has a section named after it, `agent.<program>`, holding `command`, `model` and `max_steps`; any other key there is an option the program defines, such as `reads_hooks_and_skills` for Claude Code (see [the agent page](/agent/)). A row with no kind beside it is a group holding the keys under it.

Two things are not settings and are not in here: which files count as notes, and what a vault's own folder is called. Both are asked for on the command line, by [`numen-cli`](/cli/#where-it-puts-things).

<!-- BEGIN AUTOGEN -->
| | | |
| --- | --- | --- |
| `v` | a number | the shape of the file. Nothing reads it yet, and it is written so that the day a section changes shape there is something to tell the two apart. |
| `plugins` |  | the configuration for modular plugins, keyed by plugin ID. |

### `appearance`

How the window is drawn.

| | | |
| --- | --- | --- |
| `interface_scale` | a number | how large the window is drawn: its chrome, its controls, the spacing between them and the type in them. |
| `text_scale` | a number | how large the text a person reads is set: a note, a book, an answer, the editor. |
| `mode` | text | which half of a colour pair the window takes: `system`, `light` or `dark`. |
| `theme` | text | the stylesheet the window wears, named by its origin and its filename: `preset:dracula` ships here, `mine:dracula` is the person's file. |
| `nest_headings_under_notes` | yes or no | whether a node in the plex nests the headings of its note under the box. |
| `nested_heading_limit` | a number | how many of those headings stand under a node at once, the rest being reached by scrolling. |

### `indexing`

How a vault is made searchable.

| | | |
| --- | --- | --- |
| `embedding` |  | which model turns text into vectors, and how it is reached. |
| `embedding.model` |  | what a vector is: everything that decides the space it lands in. |
| `embedding.model.name` | text |  |
| `embedding.model.dimensions` | a number |  |
| `embedding.model.max_tokens` | a number |  |
| `embedding.model.pooling` | text |  |
| `embedding.indexing` |  | where a vector is made: on this machine, or by a service. |
| `embedding.indexing.use` | text |  |
| `embedding.query` |  | where a vector is made: on this machine, or by a service. |
| `embedding.query.use` | text |  |
| `embedding.floor` | a number |  |
| `recognition` |  | how a scanned document is read when a person asks for it. |
| `recognition.recognise` |  | names the OCR model that reads scanned pages. |
| `recognition.recognise.name` | text |  |
| `recognition.proofread` |  | names the profile a recognition is put right at. |
| `recognition.proofread.with` | text | the profile, by the name the profiles carry it under. |
| `recognition.proofread.automatically` | yes or no | whether a text already written down is put right without anybody asking for it. |
| `proofreading` |  | what puts a recognition right. Naming no profile here is naming no proofreader, and a recognition is used as it was read. |
| `proofreading.profiles` |  |  |

### `agent`

Which agent answers in the panel, and what it may reach.

| | | |
| --- | --- | --- |
| `use` | text | names the program that answers. Absent, the program the providers mark as the default answers. |
| `serve_tools` | yes or no | puts the tools on a port, which is how an agent a person runs themselves reaches this vault. |

### `importing`

How an address a link note points at is reached.

| | | |
| --- | --- | --- |
| `captions` | a list of words | the languages published words are preferred in, best first. |
| `automatic_captions` | yes or no | whether words a machine wrote count where a person published none. |
| `copy_max_size_mb` | a number | maximum copy size in megabytes. Above this limit, download requests fail with the exceeded size and no data is fetched. |
| `copies_to_vault` | yes or no | whether a copy is kept beside the note as a file of the person's own. |

### `naming`

How a note's title and the name of its file are held together.

| | | |
| --- | --- | --- |
| `sync_title_and_filename` | yes or no | whether renaming either of the two brings the other into line. |

### `performance`

The workload and resource calibration profile.

| | | |
| --- | --- | --- |
| `profile` | text | the active preset. |
| `custom` |  | holds user-defined overrides when `profile` is ProfileCustom. |
| `custom.embedding_batch_size` | a number |  |
| `custom.ocr_threads` | a number |  |
| `custom.ocr_sessions` | a number |  |
| `custom.llm_concurrency` | a number |  |

### `ai.default`

The agent and the model a plugin naming no agent is answered by, and how many questions one agent answers at once.

| | | |
| --- | --- | --- |
| `agent` | text | the id of the agent plugin that answers. Empty is the provider with the lowest id. |
| `model` | text | the model the agent answers with. Empty leaves it to the agent. |
| `concurrency` | a number | how many questions one agent answers at once. |
<!-- END AUTOGEN -->
