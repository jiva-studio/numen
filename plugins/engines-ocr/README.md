# Document OCR

Recognizes text in scanned PDFs and images on your computer so their contents can be searched, read by the agent, and highlighted when opened from search results or links.

<!-- more -->

## What You Can Do

- **Make scans searchable**: Run **Recognize text** from a PDF or image file's right-click menu in the file tree, or turn on **Recognize automatically** to recognize new files when they are added to the vault.
- **Open matched passages on scanned pages**: Stores each recognized line with its page number and bounding box in a `recognition` text version, so search results and agent links open a scanned PDF at that page and highlight the passage on the page.
- **Resume after closing**: Saves progress every eight pages while a document is processed, so an interrupted run continues from the last saved batch.
- **Ask the agent to recognize a scan**: The agent in the chat panel can start the recognition of a scan with the tool `recognition_create` and read the text once it is recognized.

## Settings

- **Model**: Choose the recognition model that matches your documents (`PP-OCRv6 tiny`, `PP-OCRv6 small`, `PP-OCRv5 Latin`, `PP-OCRv5 East Slavic`, `PP-OCRv5 Cyrillic`, `PP-OCRv5 Devanagari`, or `PP-OCRv5 Greek`), or choose **Custom model** to use your own ONNX **Model file** and **Character dictionary**.
- **Scan resolution**: Resolution a page is rendered at before recognition (`150 dpi`, `200 dpi`, `300 dpi` by default, `400 dpi`, or `600 dpi`).
- **Recognize automatically**: Recognize the text of every PDF and image in the vault that has none (`false` by default).
