# Document OCR Plugin

Optical character recognition using PaddleOCR and ONNX Runtime.

## Features

- Performs document layout analysis and text recognition on scanned pages.
- Serves `numen.v1.RecognitionService`.
- Configurable scan DPI.
- Activates on `onFile:*.pdf` and `onCommand:ocr.recognize`.
