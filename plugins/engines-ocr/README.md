# Document OCR

Reads text from scanned PDFs and images so you can search, select, quote, and copy text from scans just like regular digital books.

<!-- more -->

## What You Can Do

- **Make scans searchable**: Automatically recognizes pages in scanned PDFs that don't have selectable text yet, or run **Recognise text** from a file's right-click menu at any time.
- **Recognise over selectable text**: **Recognise text** keeps the text a PDF already has. Run **Recognise text over the text layer** to replace it with a recognition.
- **Select and quote from scanned pages**: Once recognized, you can highlight text right on the scanned page, copy quotes with links, and find the document in search.
- **Resume after closing**: Long books are saved every few pages while they are processed, so progress is kept if you close the app.

## Settings

- **Model**: **Multilingual** (the default) reads every line with the Latin, Cyrillic, Greek and PP-OCRv6 tiny models and keeps the most confident reading. Choose one script instead (such as Latin, East Slavic, Cyrillic, Devanagari, Greek, or Chinese/English) to read each line once, or **Custom model** to use your own model file. The command palette offers the choice as `Document OCR: Model`.
- **Scan resolution**: Higher resolution (such as `400 dpi` or `600 dpi`) helps read small or faded print, while `300 dpi` is a fast everyday default.
