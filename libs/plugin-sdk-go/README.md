# Numen Go Plugin SDK

Backend server lifecycle SDK for native Numen plugins.

## Modules

The SDK is a base module and a feature module in each folder that has a `go.mod`. A plugin requires the base module and the feature modules it imports, each with a `replace` to its folder. A feature module requires the base module at most and no other feature module.

| Module | Packages | Used by |
|---|---|---|
| `plugin-sdk-go` | `server`, `ai`, `tools`, `hostcall`, `vault` | every native plugin; `ai` by proofreading; `tools` by a plugin that offers an agent tools; `hostcall` by the clients of the host; `vault` by readers and transcription |
| `plugin-sdk-go/host` | `host`, `host/hosttest` | a plugin that works the vault through the host |
| `plugin-sdk-go/agent` | `agent/agentrun`, `agent/reasoning`, `agent/toolcall` | agent, agent-acp, agent-antigravity, agent-claude, agent-codex |
| `plugin-sdk-go/onnx` | `onnx` | embeddings, ocr, transcription |

`make lib_plugin_sdk_go_lint`, `lib_plugin_sdk_go_build` and `lib_plugin_sdk_go_test` run over every module.

## Capabilities

- Automatic CLI flag parsing (`-port`, `-token`) with `NUMEN_PLUGIN_PORT` and `NUMEN_PLUGIN_TOKEN` environment variable fallbacks.
- Host handshake signaling via `NUMEN_READY port=<port>`.
- Connect-RPC Bearer token authentication interceptor for unary and streaming RPCs.
- Graceful lifecycle management (`Ping`, `Configure`, `Shutdown`) without unowned goroutines or sleep synchronization.
- `Configure` is how a plugin receives its settings and its secrets. `ReadSecret(values, name)` returns the key the host handed over for a secret the manifest declares and whether the call carries it; an empty key means the host holds none. `SecretAPIKey` is the name of the `apiKey` secret.
- `WithEventHandler(fn)` mounts `PluginEventService`, so the host hands the plugin each event its manifest's `activationEvents` subscribe to. An error the handler returns is answered as the plugin's error of the event, with its text as the reason. A server without one takes no events.
- Process supervision hooks (stdin EOF detection, OS signals).
- `NewLinkedHandler(vars, mount)` is the handler a host that links the plugin into its own process serves it through: a server with the services `mount` registers on it, which takes its token from the variables the host gives the plugin and reads no flag. `Server.OnClose(fn)` adds what runs when `Serve` returns, or, for a linked server, when the host shuts the plugin down.
- `agentrun`: `Discover` finds a program on the path or in the standard and program places; `Locate` finds an agent program on the path or in the places it is installed to; `CreateTempDir` creates a temporary folder with a clean-up function; the registry of the runs an agent plugin has started and the session each conversation is on, and the supervision that ends a child and everything it started. `Start` runs a child that writes one JSON event per line and returns the work that reads it, `ForwardResponses` sends the work to an `AskAgent` stream, and `Service` answers `RegisterTool`, `ExtendSystemPrompt`, `GetAgentTools` and `FinishConversation` for a plugin that embeds it.
- `ai.Complete(ctx, question)` asks the host for a model completion (needs the `ai:complete` permission); the host answers from the agent the plugin's `agent` setting names, else the default, and returns `ai.ErrNoConsent` until the person lets the plugin ask. The host endpoint comes from `NUMEN_HOST_PORT` and the token from `NUMEN_PLUGIN_TOKEN`.
- `agentrun.HasTools(req)` is whether a task names a tool of the vault: a task naming none is a single answer, and the program is given no tool. `agentrun.WritePrompt(instructions, prompt)` puts the instructions before the prompt for a program that takes no system prompt.
- `tools.Registry` is the tools a plugin offers an agent, served as `PluginToolService` once `Mount`ed (the manifest declares the `tools` capability). `tools.Add(registry, spec, run)` adds a tool from a function of a struct of arguments that answers with a struct: the schemas an agent is told come from the two structs, an error the function returns is a tool that ran and failed, and `spec` names the tool, its effect (`Reads` needs `vault:read`, `Writes` needs `vault:write`), the servers it is offered on and how a call is shown. The host lists the tools of every running plugin that declares the capability and holds each to the permission its effect needs.
- `host.Client` works the vault the window is showing through `HostVaultService`: files with their stats (`Write` returns `ErrStale` for a file that moved past its baseline), the notes, types and links the index holds, its cache and a watch. It returns the `notes/domain` types. `host.StoreClient` works the plugin's own folder of a vault's store, `.numen/<plugin id>/`, through `HostStoreService`, behind the `host.Store` interface. `hosttest.Vault` is a vault in memory that answers the same `host.Vault` interface for tests, and its `Store()` the `host.Store` one. `hostcall` reads the host endpoint and the token from the environment and presents the token on unary and streaming calls.
- `host.Texts` is the text versions of the vault the host shows: `PutVersion` reports a version of the plugin (who made it, from the file or from another version, its format and the hashes of its input and of itself), `ListVersions` names the live one, `ReadText` reads any version a page at a time through the host and `ReadWholeText` joins the pages. `With(vault)` works the texts of the vault a request names, as `host.Store.With` does. `host.TextClient` works it through `HostTextService`. A plugin declares `texts` in its manifest to put text versions. Where `texts` names `contentTypes`, the host asks the plugin for the text of a file of those types that has none, and the plugin serves a `host.TextProvider` through `NewTextProviderHandler`; a plugin that makes its text when a command runs names none. `host.DeleteText` removes a version of the plugin and its file, and `command.NewDeleteTextRun` is the run of a command that deletes the version the host names in `version-id`. `hosttest.Texts` and `hosttest.MemoryProvider` are both in memory, for tests.
- `toolcall.Describe(words, name, call, args)` is a call of a vault tool as a person is told about it, read by the `ToolDeclaration`s of the task; `toolcall.ReadPartialField` reads a field of arguments still arriving.
- The `onnx` package reads what the host sets for a plugin that declares `runtimes: ["onnxruntime"]`: `NUMEN_ONNX_RUNTIME_LIB`, the library to open sessions on, and `NUMEN_ONNX_PROVIDER`, the execution provider. `ResolveLibrary(configured, dir)` returns the library a plugin opens: its own `runtime` setting, else the library `FindLibrary` finds in the models folder its settings name, else the host's library. `FindInstalled` finds the C++ runtime the library is linked against on the loader path. It also holds what every ONNX plugin shares: the `Provider` and `ComputeUnits` settings and their parsing, the default thread count, the CoreML cache folder and options, `Open`, the one engine of the `onnxruntime_purego` binding a process opens, and `NewSessionOptions` and `NewSession`, which enable the execution provider with the CPU fallback and open a model. The `onnx` module is the one SDK module that links the binding, and only the ONNX plugins require it.

## Vault paths

- `vault.ValidatePath` returns the clean path of a file inside the vault named by `NUMEN_VAULT_PATH`, and refuses an empty path, a traversal and a path outside the vault.
