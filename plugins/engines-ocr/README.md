# Document OCR Plugin

Recognises the text of scanned pages and scanned PDFs with PaddleOCR and ONNX Runtime.

## Features

- Declares a text input of scans and PDFs (`.pdf`, `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`) and serves `numen.v1.TextProviderService`: the text it recognises is a positioned text version, page by page with the rectangles each line was read from.
- Runs on its own on a file of its types that has no live text version and whose pages carry no text of their own; a person asks for any file from its menu or with the `recognise` command (`numen-cli recognise <vault> <file>`).
- Waits its turn in the host's queue, a person's request before work nobody asked for, and shows its progress as a task.
- Keeps each recognition in its own folder of the vault's store, `.numen/md.numen.engines.ocr/<hash>.jsonl`, in the common JSON Lines format of text versions. A run in progress keeps the pages read so far in `<hash>.progress.json`, takes a run stopped part way up again where it stopped, and deletes that file once the version is put. A file is recognised again on its own when its content changes or its version is deleted; a person's request always reads it again.
- Runs a mock recogniser that reads one fixed text on every page when the host runs on mock models (`NUMEN_MOCK_ML=true`).
- Declares `onnxruntime` as a `runtime` dependency; the host provisions it and launches the plugin with `NUMEN_ONNX_RUNTIME_LIB`.
- A model setting: the recognition models the plugin provisions, each a PaddleOCR mobile recognition model in the RapidOCR ONNX export at a pinned revision, carrying its alphabet in its metadata, and declared as the dependency its option provisions. PP-OCRv6 tiny (the default: Chinese, English and Latin-script languages) and small (adds Japanese) read one model for several scripts; the PP-OCRv5 Latin, East Slavic (Russian, Ukrainian, Belarusian), Cyrillic, Devanagari and Greek models read one script each and English. A model reads the languages of its script and cannot be given one, so the languages are said under the model.
- A custom model: an ONNX recognition model file and, for a model that carries no alphabet, a character dictionary (UTF-8, one character per line, the PaddleOCR format); both are shown only while the custom model is chosen, and its languages are unknown.
- A scan resolution of 150, 200, 300, 400 or 600 dpi.
