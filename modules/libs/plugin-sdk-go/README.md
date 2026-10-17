# Numen Go Plugin SDK

Backend server lifecycle SDK for native Numen plugins.

## Capabilities

- Automatic CLI flag parsing (`-port`, `-token`) with `NUMEN_PLUGIN_PORT` and `NUMEN_PLUGIN_TOKEN` environment variable fallbacks.
- Host handshake signaling via `NUMEN_READY port=<port>`.
- Connect-RPC Bearer token authentication interceptor for unary and streaming RPCs.
- Graceful lifecycle management (`Ping`, `Configure`, `Shutdown`) without unowned goroutines or sleep synchronization.
- Process supervision hooks (stdin EOF detection, OS signals).

## Models

The `models` package is the one place plugins keep and fetch models.

- `GetHome` is the folder every plugin shares: `NUMEN_MODELS_DIR`, or `numen/models` under the platform cache directory.
- `Fetch` downloads over https into that folder, checks the bytes against the pinned sha256 while they arrive, and puts the file in place only when they match. A file with no pin is fetched as named.
- `NUMEN_OFFLINE=1` forbids every fetch, and `NUMEN_MODELS_QUOTA` caps the bytes the folder holds (32 GiB by default).
- A plugin that downloads declares `network:outbound` in its manifest.
