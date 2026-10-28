# LLM Proofreading

Cleans up spelling, punctuation, and recognition errors in audio transcripts and scanned documents using an AI model—without losing timestamps or page positions.

<!-- more -->

## What You Can Do

- **Fix transcription and OCR mistakes**: Click **Proofread** in the Media Player or right-click a transcribed recording or recognized PDF in the sidebar to produce a clean, readable version.
- **Keep original versions safe**: Proofreading creates a new text version alongside the original transcript or scan recognition, so you can switch back and forth at any time.

## Settings

- **Provider**:
  - **Agent** *(default)*: Uses one of your configured AI assistants in Numen.
  - **OpenRouter / OpenAI / Custom service**: Connects directly to an online model service to proofread multiple sections in parallel for faster results and lower cost.
- **Batch size**: How many lines are proofread together at once (`40` by default).
- **Overlap**: Extra surrounding lines included for context between batches (`4` by default).
- **In-flight requests**: How many batches are proofread in parallel (`4` by default).
- **Max edit distance**: Safety limit (`0.3` by default) that rejects a line if the AI rewrites too much of the original wording.
