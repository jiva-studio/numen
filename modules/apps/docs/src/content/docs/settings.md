---
title: Settings
description: What you can change from the window, and the one file that holds the rest.
---

Most of what you would want to change is in the commands — <kbd>Ctrl</kbd>/<kbd>⌘</kbd>
<kbd>P</kbd>, under **This window**:

| | |
| --- | --- |
| **Change the theme** | the palette the window is drawn in. See [themes](/themes/). |
| **Light or dark** | follow the system, or pin one. A theme published in one half only leaves nothing to choose here. |
| **Interface size** | how large the window is drawn: fields, buttons, spacing, type. |
| **Reading font size** | how large the text you read is set, over the size above. |

Choosing one writes it down. There is no separate save.

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
    "text_scale": 1
  }
}
```

## Search by meaning

The third band of [search](/finding/) needs a model to read your notes with. Out of the box a
small one is fetched to your machine on first use and runs there — no key, no account, nothing
sent anywhere.

**To turn it off entirely.** Searching is then names and words alone, and nothing is ever
fetched:

```json
{ "indexing": { "embedding": { "indexing": { "use": "" } } } }
```

**To have a service do it instead.** Anything that speaks the usual embeddings request will do.
Filling the index is the expensive half, and a service does in an hour what a laptop does in a
day:

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

The key is read from the environment variable you name, and numen never writes a key back into
this file.

Choosing another model means the vault is read again under it. Nothing is thrown away: setting
the old one back finds the old work where it was.

## The agent

`agent` is which assistant answers in the panel and how it is started — see
[the agent](/agent/).

## The rest of the file

There is more in it: how a scanned page is read, how near a passage must stand to a question to
count as an answer, which model corrects a reading. Every one has a default that works, and
each was measured against something particular. Leave them as they are unless you have a reason
of your own.
