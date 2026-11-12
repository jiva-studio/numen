# Writing a provider

A provider is a plugin that carries out operations for the connections of the types it declares. The core does the rest: it keeps the connections and their keys, reads the rows of the plugin that asks, tries them in order, and tells the person when one is set aside. A provider holds no state and no key between calls.

## Declare the types

The manifest lists the types under `contributes.connectionTypes` (see [the manifest fields](README.md#manifest-fields)), declares `numen.v1.ProviderService` under `services`, and has a `backend`. `operations` names the operations a type serves, lowercase words joined by dots such as `llm.chat`; the core knows them as names only. `rowFields` are the values a row that names a connection of the type holds besides the connection, such as the model: the core hands them to the provider as the options of the call, and a row field with `optionsFrom` lists its values through that operation of the type.

A manifest may give operations a title under `contributes.operations`, a list of `{ "name": "acp.task", "title": "run agent tasks" }`: the title is a lowercase verb phrase, and the settings page says "This type cannot run agent tasks" of a connection whose type does not serve the operation. The core passes the titles of every enabled plugin with the connection types, the first declaration of a name in the order of the plugin ids; an operation no plugin titles is shown by its name.

## Contracts

The core carries every operation as one generic call: the name, the connection (fields, secrets, row options) and an opaque payload, answered by data frames and one done or one failure. The shapes of the payloads and frames are contracts in the SDK, as protobuf messages with a typed client and a typed provider adapter: `plugin/sdk-go/contracts/llm` (`llm.complete`, `llm.chat`, `llm.embed`, `llm.profile`, and the option lists `llm.models` and `llm.reasoning`) and `plugin/sdk-go/contracts/acp` (`acp.task`, a prompt turn of the Agent Client Protocol). The TypeScript messages and clients are in `@numen/plugin-sdk/contracts/llm` and `@numen/plugin-sdk/contracts/acp`. The core and the core protocol never import a contract.

## Fill the roles

`plugin/sdk-go/connection` has `Provider`, a map from the name of an operation to an `Operation`, and `connection.Mount(s, provider)` serves it on the plugin server; an operation the map does not hold answers as not served. A contract fills the map: `llm.Provide(nil, llm.Roles{Completer, Chatter, Embedder, Profiler, Models, Reasoning})` adds the operations of the roles that are not nil, and `acp.Provide(provider, runner)` adds `acp.task`. An option list is `connection.ServeOptions`, which answers `numen.v1.Options`.

Every operation receives a `Call`: the connection id, the type, the fields that are not secret with the fixed values of the type, the secret fields and the options of the row. The core reads the secrets once for the call, from a snapshot of the connection, and refuses the call if the connection was edited meanwhile. `Secret` prints, logs and encodes without its value; read it from the `Value` field, send it to the service and keep it nowhere. An option such as the model is what the person typed or chose: do not check it against the list of the service.

A failure before the first data frame lets the core try the next row; after the first frame the core passes the failure to the plugin and tries no other row, so a provider ends a broken stream with a failure and never with a done. The provider ends the request to the service when `ctx` ends.

## Report failures

A call returns a result, one of two failures, or the error of its context. A failure carries a reason from a fixed list, and the core never sees an HTTP code or a text from the service.

| Failure      | Meaning                          | Constructor                           |
| ------------ | -------------------------------- | ------------------------------------- |
| try the next | this connection cannot serve now | `NewTryNextError(reason, retryAfter)` |
| stop         | the request itself is wrong      | `NewStopError(reason)`                |

A try-next failure is a rejected key (401, 403: `ReasonRejectedKey`), a payment required (402: `ReasonPaymentRequired`), a rate limit (429: `ReasonRateLimited`, with its Retry-After as `retryAfter`), and a service that cannot be reached, is too slow or answers 5xx (`ReasonUnavailable`). A stop is a missing model (404) or any other 4xx (`ReasonInvalidRequest`). An error that is neither is a stop of reason unavailable, and its text does not leave the plugin. A cancelled call returns `ctx.Err()`, which travels as the status code `Canceled`: the outbound request of the provider carries the same `ctx`, so a cancelled call ends the request to the service.

The core retries an unavailable service once. After that it sets the connection aside for 60 seconds, or for the Retry-After the service gave, at most 10 minutes. After a rejected key or a payment required it stays aside until the connection is edited. The window tells the person when a connection was set aside and the next row answered, once for each connection and reason. When every row fails, the plugin that asked gets an error that names each connection and its reason.

A provider checks the address it resolves to before it sends a key: the core checks the address as typed and resolves no host name.

## Ask for an operation

A plugin that needs an operation does not call a provider. `connection.NewClient`, `NewClientFromEnv` or `NewClientFromVars` returns a `Client`, a `connection.Caller` whose `Call(ctx, operation, payload, priority)` returns the frames of the call; the contracts wrap it, such as `llm.Complete`, `llm.Embed`, `llm.GetProfile` and `llm.Chat`, whose `ChatStream` is read with `Next` and `Frame` until `Next` returns false, then `Err` and `Finish`. Close a stream when done. The request names no connection and no key: the core takes the connections from the rows of the setting of type `connections` of the plugin whose `operations` list the operation, in the order the person set, and a plugin reaches no operation none of its settings lists. The errors are `connection.ErrNoConnection` (no row serves the operation), `*connection.AllFailedError` (a failure after the first frame is one with the one connection that failed) and `*connection.StoppedError`; use `errors.Is` and `errors.As`.

### Priority

A call carries a `Priority`: `connection.PriorityInteractive` for a person's action (a search, a check of a line) and `connection.PriorityBackground` for work that serves no person's action (indexing). `llm.GetProfile` is background.

### Queue of a connection

A connection runs at most `parallelRequests` calls at once: the value stored on the connection, else the `default` of the field in the type, else no limit when the type declares none. Every plugin that uses the connection shares the limit. The calls over the limit wait in two lists, interactive and background, each in order of arrival; when a slot frees, the head of the interactive list goes first. A call waits at most 60 seconds when it is interactive and 10 minutes when it is background. A wait that ends is "try the next" with the reason `busy` and sets no cooldown, so the core goes to the next row. A call whose context ends leaves the queue at once; a running call that is cancelled frees its slot when the provider returns.

### From TypeScript

A plugin view gets `context.work.connections`, a `ConnectionHost` whose `call(operation, payload, options)` is an async generator of the data frames that returns the done payload or a `ConnectionError` with `error` (`'no_connection'`, `'all_failed'`, `'stopped'` or `'canceled'`), `message` and `failures` (`connectionName` and `reason` for each row). The contract modules wrap it: `complete`, `embed`, `getProfile` and `chat` of `@numen/plugin-sdk/contracts/llm`. The priority is `options.priority` (`'interactive'` or `'background'`) and `options.signal` cancels the call. The calls go to `ConnectionCallService` of the window with the id of the plugin; the endpoint plugin processes reach with their token does not mount it, and the service refuses a request that carries a token.

## Agent programs as providers

`agents-acp` runs agent programs over the Agent Client Protocol and provides four types: `claude-code` (Claude Code through its adapter `claude-code-acp` or `claude-agent-acp`), `codex` (`codex-acp`), `antigravity` (`agy acp`) and `acp`, any program that speaks the protocol. Each serves `llm.complete`, `llm.models`, `llm.reasoning` and `acp.task`. The field `program` is the executable to start: required for `acp`, and for the others empty finds the known binary on the path. The model and the reasoning level are chosen in the row. Every type has the field `parallelRequests`, from 1 to 8 with the default 1, and the plugin has the setting `requestTimeout`, the seconds to wait for the program, 120 when it has no value. `agents-numen` provides the type `numen` with the operation `acp.task`: its own loop over `llm.chat` of its own row.

An `llm.complete` call runs the program once, in an empty temporary folder, with no file system, no terminal and no server of tools offered and every permission request refused. A program that cannot be found, and a run that outlasts the timeout, are try-next failures (`ReasonUnavailable`); any other failed run, and an empty answer, is a stop. The text the program wrote never leaves the plugin.

An `acp.task` call is one prompt turn. The provider lists the tools of the host its permissions allow, serves them to the program on an MCP endpoint of its own on loopback behind a token, hands each call the `tool_context` and the `vault_id` of the request, and grants the program's permission requests for those tools alone. Claude Code starts with its own tools off except the web search. Turns of one `conversation` continue one session while the provider keeps it: at most 8, and one idle for 30 minutes is closed. The frames follow the conventions of `contracts/acp`: the title of a tool call is the title of the listed tool, `content[0]` what it is about, `locations[0].path` the note it works on.

The plugins run in desktop windows alone (`targets`), so their types are not offered on a phone.
