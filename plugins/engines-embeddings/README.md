# Vector Embeddings Plugin

Vector embeddings engine for semantic search.

## Features

- Generates text embeddings using local ONNX Runtime models or remote service APIs.
- Serves `numen.v1.EmbeddingService`.
- A provider setting: Local runs BGE-M3 on this machine; OpenRouter, OpenAI and a custom OpenAI-compatible service are asked over the network. The model and the API key are shown, and handed to the backend, only for a remote provider, and the service URL only for the custom one; the settings of the other providers keep their values.
- Declares `onnxruntime` as a `runtime` dependency; the host provisions it and launches the plugin with `NUMEN_ONNX_RUNTIME_LIB`.
