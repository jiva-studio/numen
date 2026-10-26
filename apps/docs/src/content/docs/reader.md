---
title: Scanned books
description: How a scan of paper becomes text you can search, and the settings that decide how it is read.
---

A PDF that is scans of paper carries no text. The Document OCR plugin reads its pages here, on your machine: three small models, one that divides a page into its parts, one that finds the lines in a part, one that reads what a line says. Nothing is uploaded and no account is involved.

It reads a scan of its own accord: a `.pdf`, `.png`, `.jpg`, `.jpeg`, `.gif` or `.webp` whose pages carry no text and that has no text yet. You can also ask for it — **Recognise the text with Document OCR** in the [commands](/commands/), under **File**, or ask the [agent](/agent/) to read it.

What comes out is kept in the OCR plugin's folder inside the vault's own `.numen/` folder, `.numen/md.numen.engines.ocr/`, one file a document.

The three models are the plugin's own dependencies, which numen fetches for it.

## What is worth changing

The settings are the Document OCR plugin's, on its page in the [settings](/settings/), and in the settings file under `plugins` → `md.numen.engines.ocr`.

| | |
| --- | --- |
| `dpi` | what a page is rendered at before it is read. 300, from 150 to 600. Raising it reads finer print and costs time. |
| `language` | which recognition model reads the lines: `auto`, `cyrillic`, `latin` or `ch`. `auto` where you name none. |
| `custom_model_path` | a recognition model of your own, as an ONNX file, read in place of the one the plugin ships with. |
| `custom_dict_path` | the character dictionary that model reads with. |

Changing one of them takes effect the next time the plugin reads a page.

## Correcting what OCR read

The text OCR read off the pages can be corrected afterwards by a model, through the LLM Proofreading plugin. **Nothing does this by default**: it runs when you ask for it, with **Proofread the text with LLM Proofreading** in the [commands](/commands/), and until then the text is used exactly as it was read, with no key and no network.

```json
{
  "ai": {
    "default": { "agent": "claude", "model": "haiku" }
  },
  "plugins": {
    "md.numen.tools.proofreading": {
      "batch_size": 40,
      "max_edit_distance": 0.3
    }
  }
}
```

A recognition is asked about a page a request. The corrected text is a new version beside the one it was made from, and deleting it — **Delete the text LLM Proofreading made** — brings that one back.

The model that answers is the one the proofreading plugin's own **Agent** setting names, on its page of the settings; left at "Default", it is the one under `ai.default`: the agent, the model, and how many questions it answers at once. The `claude` command line you already have installed is one agent; a key belongs to the agent that holds it, and you enter it in the settings of that agent's plugin. The first time proofreading asks a model, the settings ask you whether it may. With no agent to answer, proofreading says that nothing is available; while you have not allowed it, the proofreading in the list of what is being done says "AI access not allowed — allow it in Settings → AI". Either way the text stays as it was read.

`max_edit_distance` is the safety catch: how far a correction may move a line's letters and still count as a correction, taken as the Levenshtein distance between them as a share of the longer of the two. Over 931 corrections of one book, every correction further apart than 0.3 was damage — text dragged in from the next line, or one corrected word standing in for a whole line — and everything below it was a genuine fix. A correction further apart than this is dropped, and that line is left as it was read. It is 0.3 where you name nothing.

The settings of the file itself are on [every setting](/reference/).
