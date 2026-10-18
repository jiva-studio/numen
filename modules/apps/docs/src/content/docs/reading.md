---
title: Scanned books
description: How a scan of paper becomes text you can search, and the settings that decide how it is read.
---

A PDF that is scans of paper carries no text. Right-click it in the [files](/files/) and choose **Recognise** — or ask the [agent](/agent/) to read it — and the pages are read here, on your machine: three small models, one that divides a page into its parts, one that finds the lines in a part, one that reads what a line says. Nothing is uploaded and no account is involved.

What comes out is kept in the vault's own `.numen/` folder, named after the bytes of the document, so a book that moves keeps its recognition and two copies of one book share it.

The models are fetched the first time a recognition is asked for, and the corner of the window shows how far that has got.

Everything here is `indexing.recognition` in [the settings file](/settings/).

## What is worth changing

Almost nothing: every number was measured, and the defaults are where the measurement said to put them. These are the three that come up.

| | |
| --- | --- |
| `recognise.dpi` | what a page is rendered at before it is read. 300. Raising it reads finer print and costs time. |
| `recognise.threads` | how many threads one model may use. 4. |
| `detect.expand` | how many pixels a found line is widened by before it is read. 18. The line a detector finds is drawn *inside* the letters, so without widening the top of every capital and the last letter of every line are cut away. |

`regions.body` says which parts of a page carry what the document says, and `regions.head` which of them open a section — where a name stands in that list is how deep the section sits. A part the model finds that `body` does not name is not read at all, which is how page numbers and running heads stay out of your search.

`dir` names a folder holding the models, and `download` turned off keeps numen from fetching anything — the two together are how a machine with no network reads a book. `runtime` names the ONNX Runtime library, for a machine whose own is the one to use.

## Correcting what OCR read

The text OCR read off the pages can be corrected afterwards by a model, through the proofreading plugin. **Nothing does this by default**: name no profile under `indexing.proofreading` and the text is used exactly as it was read, with no key and no network.

```json
{
  "ai": {
    "default": { "agent": "md.numen.agent-claude", "model": "haiku" }
  },
  "indexing": {
    "proofreading": {
      "max_edit_distance": 0.30,
      "profiles": {
        "careful": { "batch_size": 40 }
      }
    },
    "recognition": { "proofread": { "with": "careful", "automatically": true } }
  }
}
```

A profile is the size a text is cut into requests at, under a name you choose, and `recognition.proofread.with` names the one that corrects a recognition. `automatically` off leaves it to the hand: you ask for it on the book in front of you.

The model that answers is the one the proofreading plugin's own **Agent** setting names, in its section of the settings page; left at "Default", it is the one under `ai.default`: the agent, the model, and how many questions it answers at once. The `claude` command line you already have installed is one agent; a key belongs to the agent that holds it, and you enter it in the AI section of the settings page. The first time proofreading asks a model, the AI section asks you whether it may. With no agent to answer, or while you have not allowed it, proofreading says that nothing is available and the text stays as it was read.

`max_edit_distance` is the safety catch: how far a correction may move a line's letters and still count as a correction, taken as the Levenshtein distance between them as a share of the longer of the two. Over 931 corrections of one book, every correction further apart than 0.30 was damage — text dragged in from the next line, or one corrected word standing in for a whole line — and everything below it was a genuine fix. A correction further apart than this is dropped, and that line is left as it was read. It stands above the profiles because it is one threshold for the whole installation, and it is 0.30 where you name nothing.

The complete list is on [every setting](/reference/).
