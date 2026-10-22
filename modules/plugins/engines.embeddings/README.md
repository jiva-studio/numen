# Vector Embeddings Plugin

Vector embeddings engine for semantic search.

## Features

- Generates text embeddings using local ONNX Runtime models or remote service APIs.
- Serves `numen.v1.EmbeddingService`.
- Configurable provider (`local` or `service`), models, and API credentials.
- Declares `onnxruntime` as a `runtime` dependency; the host provisions it and launches the plugin with `NUMEN_ONNX_RUNTIME_LIB`.
