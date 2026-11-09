# LLM Proofreading

Cleans up spelling, punctuation, and recognition errors in audio transcripts and scanned documents using an AI model—without losing timestamps or page positions.

<!-- more -->

## What You Can Do

- **Fix transcription and OCR mistakes**: Click **Proofread** in the Media Viewer or right-click a transcribed recording or recognized PDF in the sidebar to produce a clean, readable version.
- **Keep the original versions**: Proofreading creates a new text version alongside the original transcript or scan recognition, so you can switch back and forth at any time.

## Settings

- **Connections**: Up to five connections that check the text, tried in order. Add a connection on the Connections page, then choose it here with a model. When one cannot answer, for example because its key was rejected or it ran out of credit, the next one is used. With no connection, proofreading is off.
- **Batch size**: How many lines are proofread together at once (`40` by default).
- **Overlap**: Extra surrounding lines included for context between batches (`4` by default).
- **In-flight requests**: How many batches are proofread in parallel (`4` by default).
- **Max edit distance**: Safety limit (`0.3` by default) that rejects a line if the AI rewrites too much of the original wording.
