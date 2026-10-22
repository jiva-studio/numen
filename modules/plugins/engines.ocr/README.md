# Document OCR Plugin

Optical character recognition using PaddleOCR and ONNX Runtime.

## Features

- Performs document layout analysis and text recognition on scanned pages.
- Serves `numen.v1.RecognitionService`.
- Configurable scan DPI.
- Activates on `onFile:*.pdf` and `onCommand:ocr.recognize`.
- Declares `onnxruntime` as a `runtime` dependency; the host provisions it and launches the plugin with `NUMEN_ONNX_RUNTIME_LIB`.
