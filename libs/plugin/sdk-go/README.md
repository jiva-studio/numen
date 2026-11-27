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

| Package                          | Use                                                                                                                 |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `server`, `server/servertest`    | the plugin process: lifecycle, authentication, events, the JSON API of the plugin's frontend                        |
| `hostcall`                       | the host endpoint and the token, from the environment or from the variables of a linked plugin                      |
| `connection`                     | providing the operations of a connection type, and asking the core to carry out an operation                        |
| `contracts/llm`, `llmtest`       | the messages, typed client and provider adapter of `llm.complete`, `llm.chat`, `llm.embed`, `llm.rerank` and others |
| `contracts/acp`, `acptest`       | the messages and provider adapter of `acp.task`                                                                     |
| `contracts/readerplace`          | the place a reader tab shows, as JSON in the Place of an open tab                                                   |
| `health`, `health/healthtest`    | the health items a plugin reports                                                                                   |
| `process`, `process/processtest` | the release of the process of a plugin whose work is done                                                           |
| `account`, `account/accounttest` | signing with the key of this device and the public identity, for a plugin with the permission `identity`            |
| `command`                        | the commands the manifest declares, served as `PluginCommandService`                                                |
| `tools`, `toolstest`             | the tools a plugin offers an agent, served as `PluginToolService`                                                   |
| `fileprovider`                   | the files a vault lists as absent, served as `FileProviderService`                                                  |
| `vaultprovider`                  | the remote vaults a plugin offers, served as `VaultProviderService`                                                 |
| `searchsource`                   | a search source, served as `SearchSourceService`                                                                    |
| `cache`                          | keyed values in memory and on disk, with expiration, backoff and background refresh                                 |
| `performance`                    | the performance limits supplied by the host                                                                 |
| `clock`, `clocktest`             | `clock.System` for the entry point and `clocktest.Fake` for tests                                                   |
| `rpcerror`                       | the error codes of the calls a plugin serves                                                                        |
| `plugintest`                     | message fixtures, the manifest as the host reads it, and `GrantStorage` for tests                                   |

## Server

`server.Run(mount)` runs a launched plugin. The port comes from `-port` or `NUMEN_PLUGIN_PORT`, and the token from `NUMEN_PLUGIN_TOKEN`. The server answers on loopback, writes `NUMEN_READY port=<port>` once it listens, and stops on `Shutdown`, on an OS signal or when standard input closes. `servertest.NewHandler(vars, mount)` is the handler of the mounted plugin for tests; it takes the token from `vars` and reads no flag. `Server.OnClose(fn)` adds what runs when the server stops.

- `SetConfigure` and `SetUpdateConfig` receive the settings of the plugin as a map of text.
- `server.WithEventHandler(fn)` receives each event the `activationEvents` of the manifest subscribe to. An error the handler returns is the plugin's error for the event.
- `Server.NewAPI(service)` serves the JSON methods of the plugin's frontend under a service the manifest declares in `services`. `server.Handle(api, "Method", fn)` mounts `POST /<service>/Method`, and `server.Stream` mounts a method that answers with Server-Sent Events. A function fails with `server.NewAPIError(code, message)`; any other error is `internal`. `createApi(service)` in `@numen/plugin-sdk` is the client.

## Connections

The core knows an operation by its name only; the payload and the frames are bytes of the contract the operation belongs to. See [Writing a provider](../../../docs/plugins/providers.md) and [Rows of connections](../../../docs/plugins/connection-rows.md).

- A provider is a `connection.Provider`, a map from operation name to `connection.Operation`, served with `connection.Mount(s, provider)`. `llm.Provide(provider, roles)` and `acp.Provide(provider, runner)` add the operations of a contract. An operation fails with `connection.NewTryNextError(reason, retryAfter)` when the core may try the next row, and with `connection.NewStopError(reason)` when the request itself is wrong. A `connection.Secret` shows no value when printed, logged or encoded.
- A consumer asks with a `connection.Caller`: `connection.NewClientFromEnv()`. The typed clients of a contract, such as `llm.Complete` and `llm.Chat`, take the caller. A call that no row answers ends with `connection.ErrNoConnection`, `*connection.AllFailedError` or `*connection.StoppedError`. `ListRows(ctx, operation)` lists the rows of an operation with the row fields of their types, `ListRowOptions(ctx, request)` the values of a row field of one of them, and `Prefer(row)` returns a caller that asks that row first with the values of its row fields given, or that row alone where `row.IsExclusive` is set, as `listRows`, `listRowOptions` and the option `preferred` with `isExclusive` of the TypeScript SDK do.
- In tests, `llmtest` and `acptest` ask the operations of a provider with the messages of the contract, and `llmtest.NewCaller(provider, call)` is a caller that answers from a provider with the errors of the core.

## Health

`health.NewClientFromEnv()` returns the health client. `Report(ctx, item)` puts an item under its key or replaces it, and `Clear(ctx, key)` takes it out. An `Error` item shows in the window's list of tasks and on the plugin's page, a `Warning` item on the page only. `health.OpenSettings(setting)` is the action that opens the settings of the plugin at a setting. The host refuses more than 8 keys, a text over 120 characters, a detail over 500, and the keys `configure`, `start` and `search`. `healthtest.NewHost(t)` keeps the reported items for tests.

## Account

`account.NewClient(endpoint)` returns the client of the identity of this device. `SignAsDevice(ctx, message)` returns the signature of the device key over 1 to 4096 bytes, made under the id of the plugin, and `GetPublicIdentity(ctx)` the fingerprints, the public signing key and the signed chain. Both need the permission `identity` and return `account.ErrNoIdentity` while the device holds none. `SealData(ctx, context, plain)` and `OpenData(ctx, context, sealed)` seal and open up to 32 KiB under a key of the plugin (`account.ErrCannotOpen` for bytes that do not open). `GetPendingRestore`, `SignAsRecovery` and `CompleteRestore` let a plugin join an account while the person restores from words (`account.ErrNoPendingRestore`, `account.ErrInvalidChain`). `accounttest.NewFake(t, pluginID)` serves all of them from a real identity in memory; `StartRestore(words)` makes it hold a pending restore.

## Process

In a backend mount, `s.GetHostEndpoint()` reads the linked registration or the process launch environment. Pass that endpoint to host service clients such as `process.NewClient(endpoint)`.

`process.NewClientFromEnv()` returns the client that releases the process. `Release(ctx)` tells the host the work is done: the host stops the process at once where no call into it is open, and else as soon as the last call ends; a later call starts it again. `process.ReleaseAndLog(ctx, releaser)` releases through a `process.Releaser` and logs a failure, after which the process stops at its idle limit. `connection.Mount` releases the process after an answer to `connections.detected` or an option list while no other call of the provider is open. `processtest.NewHost(t)` counts the releases for tests.

## Storage

`cache.GetPluginDir()` and `cache.GetDataDir()` return the plugin's cache and data directories. `cache.GetSharedDir()` returns the shared cache. A plugin requests a model its manifest declares with `host.NewDependencyClient(endpoint).FetchDependency(ctx, id)`, and a model file a person names by its address with `FetchModelFile(ctx, url, sha256)`. `plugintest.GrantStorage(t)` grants the storage directories in a test.

## Clock

A package declares the one-method interface `Now() time.Time` it needs and receives a clock through its constructor. The machine clock is read only in the `Now` method of a clock; the check `go/clock-port` in `tools/checks` rejects any other read.
