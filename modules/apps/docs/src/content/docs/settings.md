---
title: Settings
description: What you change from the window, where the settings file lives, and how it behaves.
---

Most of what you would want to change is in the commands — <kbd>Ctrl</kbd>/<kbd>⌘</kbd>
<kbd>P</kbd>, under **This window**:

| | |
| --- | --- |
| **Change the theme** | the palette the window is drawn in. See [themes](/themes/). |
| **Light or dark** | follow the system, or pin one. A theme published in one half only leaves nothing to choose here. |
| **Interface size** | how large the window is drawn: fields, buttons, spacing, type. |
| **Reading font size** | how large the text you read is set, over the size above. |
| **Sync title and filename** | whether renaming one renames the other. See [writing notes](/writing/#renaming). |
| **Hang the parts of a note under its node** | whether a node on the map brings out the note's headings. See [the plex](/plex/#what-is-inside-a-note). |
| **How many parts a node hangs** | how many of those headings stand at once, from one to twelve. |

Choosing one writes it down. There is no separate save.

The two sizes are multipliers over what was designed. The interface goes from **0.8 to 2** and
the text you read from **0.8 to 1.75**; hairlines, borders and focus rings stay one physical
line under either. A number outside those bounds is refused, said out loud, and left in the file
exactly as you wrote it.

## The file

Everything else lives in one file, `numen.json`, in [the folder numen keeps its own things
in](/install/#where-numen-keeps-its-own-files).

It is yours to edit. Fields you leave out keep their defaults, so a file naming one setting is a
complete file. Keys numen does not know are carried through untouched, and a file that does not
parse is never written over — numen says so and leaves it alone.

```json
{
  "appearance": {
    "mode": "system",
    "theme": "preset:numen",
    "interface_scale": 1,
    "text_scale": 1,
    "hang_parts_under_a_node": true,
    "parts_under_a_node": 6
  }
}
```

## What is in it

| | |
| --- | --- |
| `appearance` | how the window is drawn: the theme, which half of it, the two sizes, and what a node hangs. |
| `indexing.embedding` | the model that reads your notes so they can be found by meaning. See [search by meaning](/meaning/). |
| `indexing.recognition` | how a scanned book is read. See [reading scanned books](/reading/). |
| `indexing.proofreading` | what corrects a reading afterwards, and nothing by default. |
| `agent` | which assistant answers in the panel, and how it is started. See [the agent](/agent/). |
| `naming` | whether a note's title and its filename are kept as one name. See [writing notes](/writing/#renaming). |

[Every setting](/reference/) is the complete list, taken from the application itself.

A key never written back into this file is a key to a service: name the environment variable it
is read from, and it stays out of your files.
