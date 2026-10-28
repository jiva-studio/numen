# LLM Proofreading

Proofreads transcripts and OCR-recognized text with a language model, saving the cleaned text as a new text version while preserving timestamps and page coordinates.

<!-- more -->

## Providers & Settings

- **Provider**:
  - **Agent** *(default)*: Routes proofreading prompts through a configured Numen agent.
  - **OpenRouter / OpenAI / Custom service**: Sends parallel batch chat completion requests directly to an OpenAI-compatible API using the configured **API key** and **Model**.
- **Batching & safety controls**:
  - **Batch size**: Number of lines sent per model request (`40` by default).
  - **Overlap**: Context lines shared between adjacent batches (`4` by default).
  - **In-flight requests**: Number of parallel model requests executed concurrently (`4` by default).
  - **Max edit distance**: Maximum allowed normalized edit distance (`0.3` by default) to guard against model hallucinations or dropped lines.
