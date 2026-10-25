# Document OCR Plugin

Recognises the text of scanned pages and scanned PDFs with PaddleOCR and ONNX Runtime.

## Features

- Declares a text input of scans and PDFs (`.pdf`, `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`) and serves `numen.v1.TextProviderService`: the text it recognises is a positioned text version, page by page with the rectangles each line was read from.
- Runs on its own on a file of its types that has no live text version and whose pages carry no text of their own; a person asks for any file from its menu or with the `recognise` command (`numen-cli recognise <vault> <file>`).
- Waits its turn in the host's queue, a person's request before work nobody asked for, and shows its progress as a task.
- Keeps each recognition in its own folder of the vault's store, `.numen/md.numen.engines.ocr/`, and takes a run stopped part way up again where it stopped.
- Runs a mock recogniser that reads one fixed text on every page when the host runs on mock models (`NUMEN_MOCK_ML=true`).
- Declares `onnxruntime` as a `runtime` dependency; the host provisions it and launches the plugin with `NUMEN_ONNX_RUNTIME_LIB`.
