# Numen Go Plugin SDK

Backend server lifecycle SDK for native Numen plugins.

## Capabilities

- Automatic CLI flag parsing (`-port`, `-token`) with `NUMEN_PLUGIN_PORT` and `NUMEN_PLUGIN_TOKEN` environment variable fallbacks.
- Host handshake signaling via `NUMEN_READY port=<port>`.
- Connect-RPC Bearer token authentication interceptor for unary and streaming RPCs.
- Graceful lifecycle management (`Ping`, `Configure`, `Shutdown`) without unowned goroutines or sleep synchronization.
- Process supervision hooks (stdin EOF detection, OS signals).
- `ai.Complete(ctx, question)` asks the host for a model completion by purpose (needs the `ai:complete` permission); the host endpoint comes from `NUMEN_HOST_PORT` and the token from `NUMEN_PLUGIN_TOKEN`.
- `completion.Answer(ctx, question, run)` is the body of `CompletionService.Complete` for an agent plugin: it reads the question, answers the model's text, answers `UNAVAILABLE` (reason in the log) when `run` fails or sends no text, and returns a cancelled call as cancelled. `completion.RunCommand` runs a CLI with the prompt on stdin.

## Models

The `models` package is the one place plugins keep and fetch models.

- `GetHome` is the folder every plugin shares: `NUMEN_MODELS_DIR`, or `numen/models` under the application data directory.
- `Fetch` downloads over https into that folder, checks the bytes against the pinned sha256 while they arrive, and puts the file in place only when they match. A file with no pin is fetched as named.
- `NUMEN_OFFLINE=1` forbids every fetch, and `NUMEN_MODELS_QUOTA` caps the bytes the folder holds (32 GiB by default).
- A plugin that downloads declares `network:outbound` in its manifest.

## Pins and vault paths

- `models.Pins` maps a model address to its pinned sha256. `models.Locate` returns where a model file is: a path as written, a file in the places a `models.Source` names (a pinned file only when it holds its sum), or a fetch.
- `vault.ValidatePath` returns the clean path of a file inside the vault named by `NUMEN_VAULT_PATH`, and refuses an empty path, a traversal and a path outside the vault. `vault.IsTempDocument` lets a document the host put in the temp folder through.
