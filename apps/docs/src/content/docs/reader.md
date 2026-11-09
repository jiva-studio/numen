---
title: Scanned books
description: How a scan of paper becomes text you can search, and the settings that decide how it is read.
---

A PDF that is scans of paper has no text. The Document OCR plugin reads its pages here, on your machine: three small models, one that divides a page into its parts, one that finds the lines in a part, one that reads the text of a line. Nothing is uploaded and no account is involved.

It reads a scan automatically: a `.pdf`, `.png`, `.jpg`, `.jpeg`, `.gif` or `.webp` whose pages contain no text and that has no text yet. You can also start it yourself with **Recognise the text with Document OCR** in the [commands](/commands/), under **File**, or ask the [agent](/agent/) to read it.

The result is stored in the OCR plugin's folder inside the vault's own `.numen/` folder, `.numen/md.numen.engines.ocr/`, one file per document.

The three models are the plugin's own dependencies, which numen fetches for it.

## Settings you may want to change

The settings are the Document OCR plugin's, on its page in the [settings](/settings/), and in the settings file in the object `md.numen.engines.ocr`.

|                   |                                                                                                                                             |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `dpi`             | the resolution a page is rendered at before it is read. The default is 300, from 150 to 600. Raising it reads finer print but takes longer. |
| `language`        | which recognition model reads the lines: `auto`, `cyrillic`, `latin` or `ch`. `auto` is the default.                                        |
| `customModelPath` | a recognition model of your own, as an ONNX file, read in place of the one the plugin ships with.                                           |
| `customDictPath`  | the character dictionary that model uses.                                                                                                   |

Changing one of them takes effect the next time the plugin reads a page.

## Correcting what OCR read

The text OCR read from the pages can be corrected afterwards by a model, through the LLM Proofreading plugin. **Nothing does this by default**: it runs when you ask for it, with **Proofread the text with LLM Proofreading** in the [commands](/commands/), and until then the text is used exactly as it was read, with no key and no network.

```json
{
  "md.numen.tools.proofreading": {
    "connections": [{ "connection": "c1f0a9e20", "model": "google/gemini-2.5-flash" }],
    "batchSize": 40,
    "maxEditDistance": 0.3
  }
}
```

Each request to the model covers one page. The corrected text is a new version next to the one it was made from, and deleting it (**Delete the text LLM Proofreading made**) restores that one.

The model that is used is chosen on the proofreading plugin's page of the settings, in its **Connections** rows: up to five rows, each a connection you made on the Connections page, with a model and optionally a reasoning level. Numen tries the rows in order, and goes to the next one when a service rejects the key, asks for payment, limits the rate or cannot be reached. A connection can be a service such as OpenRouter or OpenAI, a server such as Ollama, or an agent on this computer such as Claude Code. The key stays with the connection and the plugin never sees it. With no row, proofreading is off: its page has a button to the Connections page, and the text stays as it was read.

`maxEditDistance` is the safety limit: how far a correction may change a line's letters and still count as a correction, taken as the Levenshtein distance between them as a share of the longer of the two. Over 931 corrections of one book, every correction with a distance above 0.3 was an error (text taken from the next line, or one corrected word replacing a whole line), and every correction below it was a genuine fix. A correction with a larger distance is dropped, and that line is left as it was read. The default is 0.3.

The settings of the file itself are on [every setting](/reference/).
