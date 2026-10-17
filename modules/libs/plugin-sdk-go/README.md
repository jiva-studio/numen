# Numen Go Plugin SDK

Backend server lifecycle SDK for native Numen plugins.

## Capabilities

- Automatic CLI flag parsing (`-port`, `-token`) with `NUMEN_PLUGIN_PORT` and `NUMEN_PLUGIN_TOKEN` environment variable fallbacks.
- Host handshake signaling via `NUMEN_READY port=<port>`.
- Connect-RPC Bearer token authentication interceptor for unary and streaming RPCs.
- Graceful lifecycle management (`Ping`, `Configure`, `Shutdown`) without unowned goroutines or sleep synchronization.
- Process supervision hooks (stdin EOF detection, OS signals).
- `agentrun`: the registry of the runs an agent plugin has started and the session each conversation is on, and the supervision that ends a child and everything it started.
