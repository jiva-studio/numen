# Document OCR

Recognizes text from scanned PDFs and images using PaddleOCR mobile ONNX models and stores positioned page text for reading and search.

<!-- more -->

## Supported Formats & Models

- **File formats**: `.pdf`, `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`.
- **Automatic & on-demand recognition**: Runs automatically in the background queue on scanned files that carry no embedded text, or on demand from the file menu (**Recognise text**).
- **Models (downloaded from Hugging Face Hub on first use)**:
  - **PP-OCRv6 Tiny** *(default)*: Chinese, English, and Latin-script languages.
  - **PP-OCRv6 Small**: Chinese, English, Japanese, and Latin-script languages.
  - **PP-OCRv5 Script Models**: Dedicated models for Latin, East Slavic (Russian, Ukrainian, Belarusian), Cyrillic, Devanagari, and Greek scripts.
  - **Custom ONNX model**: Load your own `.onnx` recognition model and optional UTF-8 character dictionary file (`.txt`, one character per line).

## Settings & Storage

- **Model**: Selects the PaddleOCR recognition model or custom ONNX weights.
- **Scan resolution**: Renders PDF pages at `150`, `200`, `300`, `400`, or `600` DPI before recognition.
- **Vault storage**: Recognized text versions are saved in `.numen/md.numen.engines.ocr/<hash>.jsonl` with resumable per-page progress checkpoints.
