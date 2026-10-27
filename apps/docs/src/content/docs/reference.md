---
title: Settings reference
description: Every key numen.json can hold, section by section, taken from the application itself.
---

Every key `numen.json` can hold. The pages under Settings say which of these are worth touching and why; this one is here so that nothing is missing.

Each heading below is a section of the file, and the keys under it are written the way they are written inside that section — `max_tokens` under `indexing.embedding` is `indexing.embedding.max_tokens` if you write it out in full.

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
| `theme` | text | the stylesheet applied to the window, named by its origin and its filename: `preset:dracula` ships here, `mine:dracula` is the person's file. |

### `indexing.embedding`

How the index keeps and asks vectors.

| | | |
| --- | --- | --- |
| `max_tokens` | a number | the input bound, in tokens, of a chunk the index keeps a vector of. |
| `floor` | a number | how near a question a passage stands to be an answer at all. |

### `agent`

Which agent answers in the panel, and what it may reach.

| | | |
| --- | --- | --- |
| `use` | text | names the program that answers. Absent, the program the providers mark as the default answers. |
| `serve_tools` | yes or no | puts the tools on a port, which is how an agent a person runs themselves reaches this vault. |

### `naming`

Whether a note's title and its filename are kept in sync.

| | | |
| --- | --- | --- |
| `sync_title_and_filename` | yes or no | whether renaming either of the two brings the other into line. |

### `performance`

The workload and resource calibration profile.

| | | |
| --- | --- | --- |
| `profile` | text | the active preset. |

### `ai.default`

The agent and the model a plugin naming no agent is answered by, and how many questions one agent answers at once.

| | | |
| --- | --- | --- |
| `agent` | text | the id of the agent plugin that answers. Empty is the provider with the lowest id. |
| `model` | text | the model the agent answers with. Empty leaves it to the agent. |
| `concurrency` | a number | how many questions one agent answers at once. |
<!-- END AUTOGEN -->
