# Numen Go Plugin SDK

The Go SDK of a native Numen plugin: the server a plugin process runs under, the clients of the host, the connections of the core, and the helpers its tests use. A plugin imports the SDK only; it imports no protobuf or Connect package of the core and nothing under `libs/core`.

## Modules

The SDK is a base module and a feature module in each folder that has a `go.mod`. A plugin requires the base module and the feature modules it imports, each with a `replace` to its folder. A feature module requires the base module at most and no other feature module.

| Module               | Provides                                                                            |
| -------------------- | ----------------------------------------------------------------------------------- |
| `plugin/sdk-go`      | the server, the connections and their contracts, tools, health, cache and the rest  |
| `plugin/sdk-go/host` | the vault, the store, the search index, the text versions and the queue of the host |
| `plugin/sdk-go/wasm` | the compilation cache of a plugin that runs WebAssembly                             |

`make lib_plugin_sdk_go_lint`, `lib_plugin_sdk_go_build` and `lib_plugin_sdk_go_test` run over every module.

## Packages

| Package                          | Use                                                                                                   |
| -------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `server`, `server/servertest`    | the plugin process: lifecycle, authentication, events, the JSON API of the plugin's frontend          |
| `hostcall`                       | the host endpoint and the token, from the environment or from the variables of a linked plugin        |
| `connection`                     | providing the operations of a connection type, and asking the core to carry out an operation          |
| `contracts/llm`, `llmtest`       | the messages, typed client and provider adapter of `llm.complete`, `llm.chat`, `llm.embed` and others |
| `contracts/acp`, `acptest`       | the messages and provider adapter of `acp.task`                                                       |
| `health`, `health/healthtest`    | the health items a plugin reports                                                                     |
| `process`, `process/processtest` | the release of the process of a plugin whose work is done                                             |
| `command`                        | the commands the manifest declares, served as `PluginCommandService`                                  |
| `tools`, `toolstest`              | the tools a plugin offers an agent, served as `PluginToolService`                                     |
| `searchsource`                   | a search source, served as `SearchSourceService`                                                      |
| `cache`                          | keyed values in memory and on disk, with expiration, backoff and background refresh                   |
| `shared`                         | the lock and the model key of the directory the host shares between plugins                           |
| `performance`                    | the performance profile of the person and `SetNice`                                                   |
| `clock`                          | `clock.System` for the entry point and `clock.Fake` for tests                                         |
| `rpcerror`                       | the error codes of the calls a plugin serves                                                          |
| `plugintest`                     | message fixtures, the manifest as the host reads it, and `GrantStorage` for tests                     |

## Server

`server.Run(mount)` runs a launched plugin. The port comes from `-port` or `NUMEN_PLUGIN_PORT`, and the token from the first line of standard input. The server answers on loopback, writes `NUMEN_READY port=<port>` once it listens, and stops on `Shutdown`, on an OS signal or when standard input closes. `server.NewLinkedHandler(vars, mount)` is the handler of a plugin the host links into its own process; it takes the token from `vars` and reads no flag. `Server.OnClose(fn)` adds what runs when the server stops.

- `SetConfigure` and `SetUpdateConfig` receive the settings of the plugin as a map of text.
- `server.WithEventHandler(fn)` receives each event the `activationEvents` of the manifest subscribe to. An error the handler returns is the plugin's error for the event.
- `Server.NewAPI(service)` serves the JSON methods of the plugin's frontend under a service the manifest declares in `services`. `server.Handle(api, "Method", fn)` mounts `POST /<service>/Method`, and `server.Stream` mounts a method that answers with Server-Sent Events. A function fails with `server.NewAPIError(code, message)`; any other error is `internal`. `createApi(service)` in `@numen/plugin-sdk` is the client.

## Connections

The core knows an operation by its name only; the payload and the frames are bytes of the contract the operation belongs to. See [Writing a provider](../../../docs/plugins/providers.md) and [Rows of connections](../../../docs/plugins/connection-rows.md).

- A provider is a `connection.Provider`, a map from operation name to `connection.Operation`, served with `connection.Mount(s, provider)`. `llm.Provide(provider, roles)` and `acp.Provide(provider, runner)` add the operations of a contract. An operation fails with `connection.NewTryNextError(reason, retryAfter)` when the core may try the next row, and with `connection.NewStopError(reason)` when the request itself is wrong. A `connection.Secret` shows no value when printed, logged or encoded.
- A consumer asks with a `connection.Caller`: `connection.NewClientFromEnv()` for a launched plugin, `connection.NewClientFromVars(vars)` for a linked one. The typed clients of a contract, such as `llm.Complete` and `llm.Chat`, take the caller. A call that no row answers ends with `connection.ErrNoConnection`, `*connection.AllFailedError` or `*connection.StoppedError`. `ListRows(ctx, operation)` lists the rows of an operation with the row fields of their types, `ListRowOptions(ctx, request)` the values of a row field of one of them, and `Prefer(row)` returns a caller that asks that row first with the values of its row fields given, or that row alone where `row.IsExclusive` is set, as `listRows`, `listRowOptions` and the option `preferred` with `isExclusive` of the TypeScript SDK do.
- In tests, `llmtest` and `acptest` ask the operations of a provider with the messages of the contract, and `llmtest.NewCaller(provider, call)` is a caller that answers from a provider with the errors of the core.

## Health

`health.NewClientFromEnv()` and `health.NewClientFromVars(vars)` return the health client. `Report(ctx, item)` puts an item under its key or replaces it, and `Clear(ctx, key)` takes it out. An `Error` item shows in the window's list of tasks and on the plugin's page, a `Warning` item on the page only. `health.OpenSettings(setting)` is the action that opens the settings of the plugin at a setting. The host refuses more than 8 keys, a text over 120 characters, a detail over 500, and the keys `configure`, `start` and `search`. `healthtest.NewHost(t)` keeps the reported items for tests.

## Process

`process.NewClientFromEnv()` returns the client that releases the process. `Release(ctx)` tells the host the work is done: the host stops the process at once where no call into it is open, and else as soon as the last call ends; a later call starts it again. `process.ReleaseAndLog(ctx, releaser)` releases through a `process.Releaser` and logs a failure, after which the process stops at its idle limit. `connection.Mount` releases the process after an answer to `connections.detected` or an option list while no other call of the provider is open. `processtest.NewHost(t)` counts the releases for tests.

## Storage

`cache.GetPluginDir()` and `cache.GetDataDir()` return the directories the host granted the plugin, from `NUMEN_PLUGIN_CACHE_DIR` and `NUMEN_PLUGIN_DATA_DIR`. `cache.GetSharedDir()` and `cache.GetSharedDataDir()` return the directories all plugins share, from `NUMEN_SHARED_CACHE_DIR` and `NUMEN_SHARED_DATA_DIR`; a plugin writes there only content named by its digest, writes it atomically and deletes nothing. `cache.GetSharedDirOrPluginDir()` and `cache.GetSharedDataDirOrPluginDataDir()` fall back to the plugin's own directory. Each returns an error where the host granted none; a plugin computes no platform path. `plugintest.GrantStorage(t)` sets all four for a test.

## Clock

A package declares the one-method interface `Now() time.Time` it needs and receives a clock through its constructor. The machine clock is read only in the `Now` method of a clock; the check `go/clock-port` in `tools/checks` rejects any other read.
