# Numen Go Plugin SDK

Backend server lifecycle SDK for native Numen plugins.

## Modules

The SDK is a base module and a feature module in each folder that has a `go.mod`. A plugin requires the base module and the feature modules it imports, each with a `replace` to its folder. A feature module requires the base module at most and no other feature module.

| Module | Packages | Used by |
|---|---|---|
| `plugin-sdk-go` | `server`, `ai` | every native plugin; `ai` by proofreading |
| `plugin-sdk-go/agent` | `agent/agentrun`, `agent/completion` | agent, agent-acp, agent-claude |
| `plugin-sdk-go/models` | `models` | embeddings, ocr, transcription |
| `plugin-sdk-go/onnx` | `onnx` | embeddings, ocr, transcription |
| `plugin-sdk-go/vault` | `vault` | reader-epub, reader-pdf |

`make lib_plugin_sdk_go_lint`, `lib_plugin_sdk_go_build` and `lib_plugin_sdk_go_test` run over every module.

## Capabilities

- Automatic CLI flag parsing (`-port`, `-token`) with `NUMEN_PLUGIN_PORT` and `NUMEN_PLUGIN_TOKEN` environment variable fallbacks.
- Host handshake signaling via `NUMEN_READY port=<port>`.
- Connect-RPC Bearer token authentication interceptor for unary and streaming RPCs.
- Graceful lifecycle management (`Ping`, `Configure`, `Shutdown`) without unowned goroutines or sleep synchronization.
- `Configure` is how a plugin receives its settings and its secrets. `ReadSecret(values, name)` returns the key the host handed over for a secret the manifest declares and whether the call carries it; an empty key means the host holds none. `SecretAPIKey` is the name of the `apiKey` secret.
- Process supervision hooks (stdin EOF detection, OS signals).
- `agentrun`: `Locate` finds an agent program on the path or in the places it is installed to; the registry of the runs an agent plugin has started and the session each conversation is on, and the supervision that ends a child and everything it started.
- `ai.Complete(ctx, question)` asks the host for a model completion by purpose (needs the `ai:complete` permission); the host endpoint comes from `NUMEN_HOST_PORT` and the token from `NUMEN_PLUGIN_TOKEN`.
- `completion.Answer(ctx, question, run)` is the body of `CompletionService.Complete` for an agent plugin: it reads the question, answers the model's text, answers `UNAVAILABLE` (reason in the log) when `run` fails or sends no text, and returns a cancelled call as cancelled. `completion.RunCommand` runs a CLI with the prompt on stdin.

## Models

The `models` package is the one place plugins keep and fetch models.

- `GetHome` is the folder every plugin shares: `NUMEN_MODELS_DIR`, or `numen/models` under the application data directory.
- `Fetch` downloads over https into that folder, checks the bytes against the pinned sha256 while they arrive, and puts the file in place only when they match. A file with no pin is fetched as named.
- `Folder` carries the settings block, folder, download switch and progress listener of a plugin; it fetches through them, lists where a model file may be beside the application, and gives the `Source` that `Locate` takes.
- The `onnx` package reads what the host sets for a plugin that declares `runtimes: ["onnxruntime"]`: `NUMEN_ONNX_RUNTIME_LIB`, the library to open sessions on, and `NUMEN_ONNX_PROVIDER`, the execution provider. `ResolveLibrary(configured, dir)` returns the library a plugin opens: its own `runtime` setting, else the library `FindLibrary` finds in the models folder its settings name, else the host's library. `FindInstalled` finds the C++ runtime the library is linked against on the loader path. It also holds what every ONNX plugin shares: the `Provider` and `ComputeUnits` settings and their parsing, the default thread count, the CoreML cache folder and options, `Open`, the one engine of the `onnxruntime_purego` binding a process opens, and `NewSessionOptions` and `NewSession`, which enable the execution provider with the CPU fallback and open a model. The `onnx` module is the one SDK module that links the binding, and only the ONNX plugins require it.
- `NUMEN_OFFLINE=1` forbids every fetch, and `NUMEN_MODELS_QUOTA` caps the bytes the folder holds (32 GiB by default).
- A plugin that downloads declares `network:outbound` in its manifest.

## Pins and vault paths

- `models.Pins` maps a model address to its pinned sha256. `models.Locate` returns where a model file is: a path as written, a file in the places a `models.Source` names (a pinned file only when it holds its sum), or a fetch.
- `vault.ValidatePath` returns the clean path of a file inside the vault named by `NUMEN_VAULT_PATH`, and refuses an empty path, a traversal and a path outside the vault. `vault.IsTempDocument` lets a document the host put in the temp folder through.
