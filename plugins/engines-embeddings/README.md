# Vector Embeddings

Generates dense vector embeddings for semantic search across notes, books, PDFs, and transcripts using local ONNX models or remote embedding APIs.

<!-- more -->

## Providers & Models

- **Local (ONNX Runtime)**: Runs on your machine using `Xenova/bge-m3` (int8 quantized, 100+ languages, 1024 dimensions, up to 8192 tokens) downloaded automatically from Hugging Face Hub on first use, or any custom Hugging Face ONNX repository (`<org>/<repo>` containing `onnx/model.onnx` and `tokenizer.json`).
- **OpenRouter**: Calls `https://openrouter.ai/api/v1/embeddings` with preset or custom model IDs (`openai/text-embedding-3-small`, `openai/text-embedding-3-large`, `baai/bge-m3`, `qwen/qwen3-embedding-8b`).
- **OpenAI**: Calls `https://api.openai.com/v1/embeddings` (`text-embedding-3-small`, `text-embedding-3-large`, `text-embedding-ada-002`).
- **Custom service**: Connects to any OpenAI-compatible `/v1/embeddings` endpoint (such as Ollama, vLLM, or Text Embeddings Inference).

## Settings

- **Provider**: Selects between local ONNX execution and remote API providers.
- **API key**: Stored in the system keychain; shown only when a remote provider is selected.
- **Service URL**: Base URL of a custom OpenAI-compatible embeddings server; shown only when **Custom service** is selected.
- **Model**: Searchable preset list for the active provider, with support for typing any custom Hugging Face repo or model identifier.
