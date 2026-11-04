---
title: Settings reference
description: Every key numen.json can hold, section by section, generated from the application itself.
---

Every key `numen.json` can hold. The pages under Settings say which of these are worth touching and why; this one lists every key.

Each heading below is a section of the file, and the keys under it are written the way they are written inside that section — `max_tokens` under `indexing.embedding` is `indexing.embedding.max_tokens` if you write it out in full.

A key you leave out keeps its default, so a file naming one setting is a complete file. A key numen does not know is kept unchanged. Under `agent`, each program an agent plugin lists has a section named after it, `agent.<program>`, holding `command` and `model`; any other key there is an option the program defines, such as `reads_hooks_and_skills` for Claude Code (see [the agent page](/agent/)). A row with no kind beside it is a group holding the keys under it.

Two things are not settings and are not in here: which files count as notes, and what a vault's own folder is called. Both are set on the command line, by [`numen-cli`](/cli/#where-it-puts-things).

<!-- BEGIN AUTOGEN -->
| | | |
| --- | --- | --- |
| `v` | a number | the format version of the file. Nothing reads it yet. |
| `plugins` |  | holds the settings of each plugin, keyed by plugin id. |

### `appearance`

Holds the theme, the mode and the sizes of the window.

| | | |
| --- | --- | --- |
| `interface_scale` | a number | the scale of the window: its chrome, its controls, the spacing between them and the type in them. |
| `text_scale` | a number | the scale of reading text: a note, a book, an answer, the editor. |
| `mode` | text | which half of a colour pair the window takes: `system`, `light` or `dark`. |
| `theme` | text | the stylesheet applied to the window, named by its origin and its identifier: `preset:numen` ships here, `mine:dracula` is the person's file, and `plugin:org.example.themes/dark` is a theme of an enabled plugin. |

### `indexing.embedding`

How notes are split into chunks and how the vectors of the chunks are stored.

| | | |
| --- | --- | --- |
| `max_tokens` | a number | the input limit, in tokens, of one chunk that gets a vector. |

### `agent`

Selects the agent the panel uses and configures how each program runs.

| | | |
| --- | --- | --- |
| `use` | text | names the program the panel uses. When absent, the program that the providers mark as default is used. |

### `naming`

Whether a note's title and its filename are kept in sync.

| | | |
| --- | --- | --- |
| `sync_title_and_filename` | yes or no | whether renaming either the title or the filename updates the other. |

### `performance`

The settings section for workload and resource profile.

| | | |
| --- | --- | --- |
| `profile` | text | the active preset. |

### `ai.default`

The agent and the model that handle the questions of a plugin whose own setting names no agent, and how many questions one agent handles at once.

| | | |
| --- | --- | --- |
| `agent` | text | the id of the agent plugin to use. Empty selects the provider with the lowest id. |
| `model` | text | the model the agent uses. Empty leaves the choice to the agent. |
| `concurrency` | a number | how many questions one agent handles at once. |
<!-- END AUTOGEN -->
