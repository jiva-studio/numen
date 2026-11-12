# Writing a provider

A provider is a plugin that carries out operations for the connections of the types it declares. The core does the rest: it keeps the connections and their keys, reads the rows of the plugin that asks, tries them in order, and tells the person when one is set aside. A provider holds no state and no key between calls.

## Declare the types

The manifest lists the types under `contributes.connectionTypes` (see [the manifest fields](README.md#manifest-fields)), declares `numen.v1.ProviderService` under `services`, and has a `backend`. `operations` names what a type carries out: `complete` and `embed` today. A type lists only the operations its provider fills with a role.

## Fill the roles

`plugin/sdk-go/connection` has one interface for each role, and a `Provider` struct that lists the roles a plugin fills. `connection.Mount(s, provider)` serves them on the plugin server; a role left nil answers as not served.

| Role            | Methods                                  | Operation                          |
| --------------- | ---------------------------------------- | ---------------------------------- |
| `Completer`     | `Complete`                               | `complete`                         |
| `Embedder`      | `Embed`                                  | `embed`                            |
| `Chatter`       | `Chat`                                   | `chat`                             |
| `ModelLister`   | `ListModels`, `ListReasoningLevels`      | the editor lists models and levels |
| `ProfileReader` | `GetModelProfile`, `ProbeEmbeddingWidth` | `embed`                            |

Every method receives a `Call`: the type, the fields that are not secret with the fixed values of the type, the secret fields, the model and the reasoning level. The core reads the secrets once for the call, from a snapshot of the connection, and refuses the call if the connection was edited meanwhile. `Secret` prints, logs and encodes without its value; read it from the `Value` field, send it to the service and keep it nowhere. The model is the name the person typed or chose: do not check it against the list of the service. `Embed` returns raw vectors, and the plugin that asks applies the prefixes of the profile.

`GetModelProfile` returns `IsKnown: false` for a model it has no profile for, and the core then calls `ProbeEmbeddingWidth`, which embeds one short text and returns the width. `MaxTokens` is 0 when the limit is not known.

`Chat` sends the events of one turn through the function `send`, in order: text deltas (never empty), the complete tool calls, then one `Finish` with the reason `FinishStop`, `FinishToolCalls` or `FinishLength`. A turn is one request to the service and the history travels with it; the messages have the roles system, user, assistant and tool, an assistant message carries its `ToolCalls`, and a tool message carries the `ToolCallID` it answers. The provider joins the pieces of a streamed tool call into one event, and reports `{}` for a call with no arguments. A failure before the first event lets the core try the next row; after the first event the core passes the failure to the plugin and tries no other row, so the provider ends a broken stream with a failure and never with a finish. The provider ends the request to the service when `ctx` ends.

The calls `SubmitJob`, `GetJobStatus` and `FetchJobResult` are in the schema for deferred requests. The core does not call them, and the SDK does not serve them.

## Report failures

A call returns a result, one of two failures, or the error of its context. A failure carries a reason from a fixed list, and the core never sees an HTTP code or a text from the service.

| Failure      | Meaning                          | Constructor                           |
| ------------ | -------------------------------- | ------------------------------------- |
| try the next | this connection cannot serve now | `NewTryNextError(reason, retryAfter)` |
| stop         | the request itself is wrong      | `NewStopError(reason)`                |

A try-next failure is a rejected key (401, 403: `ReasonKeyRejected`), a payment required (402: `ReasonPaymentRequired`), a rate limit (429: `ReasonRateLimited`, with its Retry-After as `retryAfter`), a service that cannot be reached or answers 502, 503 or 504 (`ReasonUnreachable`) and a timeout (`ReasonTimeout`). A stop is a missing model (404) or any other 4xx (`ReasonBadRequest`) and a 500 (`ReasonUnknown`). An error that is neither is a stop of unknown reason, and its text does not leave the plugin. A cancelled call returns `ctx.Err()`, which travels as the status code `Canceled`: the outbound request of the provider carries the same `ctx`, so a cancelled call ends the request to the service.

The core retries a timeout or an unreachable service once. After that it sets the connection aside for 60 seconds, or for the Retry-After the service gave, at most 10 minutes. After a rejected key or a payment required it stays aside until the connection is edited. The window tells the person when a connection was set aside and the next row answered, once for each connection and reason. When every row fails, the plugin that asked gets an error that names each connection and its reason.

A provider checks the address it resolves to before it sends a key: the core checks the address as typed and resolves no host name.

## Ask for an operation

A plugin that needs a model does not call a provider. `connection.NewClient`, `NewClientFromEnv` or `NewClientFromVars` returns a `Client` with `Complete`, `Embed`, `GetEmbeddingProfile` and `Chat`. `Chat` returns a `ChatStream`: call `Next` until it returns false, then read `Err`, which is nil after the finish, `ErrNoConnection`, `*AllFailedError` or `*RejectedError` where the core could not serve the turn, the error of the context, or `ErrStreamEnded`; a failure after the first event is an `*AllFailedError` with the one connection that failed. Close the stream when done. The request names no connection, no model and no key: the core takes the connections from the rows in the settings of the plugin, in the order the person set, and a plugin cannot use a connection that is not in its rows. The errors are `connection.ErrNoConnection` (no row serves the operation), `*connection.AllFailedError` and `*connection.RejectedError`; use `errors.Is` and `errors.As`.

A plugin compiled into the core is called through the in-process pipe, as a plugin process is called over the loopback, and the contract names no transport.

### Priority

`CompleteRequest` and `EmbedRequest` carry a `Priority`: `connection.PriorityInteractive` for a person's action (a search, a check of a line) and `connection.PriorityBackground` for work that serves no person's action (indexing). A request that sets none is background. `GetEmbeddingProfile` is background.

### Queue of a connection

A connection runs at most `parallelRequests` calls at once: the value stored on the connection, else the `default` of the field in the type, else no limit when the type declares none. Every plugin that uses the connection shares the limit. The calls over the limit wait in two lists, interactive and background, each in order of arrival; when a slot frees, the head of the interactive list goes first. A call waits at most 60 seconds when it is interactive and 10 minutes when it is background. A wait that ends is "try the next" with the reason `busy` and sets no cooldown, so the core goes to the next row. A call whose context ends leaves the queue at once; a running call that is cancelled frees its slot when the provider returns.

### From TypeScript

A plugin view gets `context.work.models`, a `ModelHost` with `complete`, `embed`, `getEmbeddingProfile` and `chat`, which is an async generator of the events of a turn that returns `null` after the finish and the `ModelError` otherwise. They take the same fields as the Go client; the priority is `options.priority` (`'interactive'` or `'background'`) and `options.signal` cancels the call. Each returns a `Result` whose `error` is a `ModelError` with `error` (`'no_connection'`, `'all_failed'`, `'rejected'` or `'canceled'`), `message` and `failures` (`connectionName` and `reason` for each row). The calls go to `ModelService` of the window with the id of the plugin; the endpoint plugin processes reach with their token does not mount `ModelService`, and the service refuses a request that carries a token.

## Agent programs as providers

The agent plugins (`agents-claude`, `agents-codex`, `agents-antigravity` and `agents-acp`) provide one type each, with the operation `complete`. A type of Claude Code, OpenAI Codex or Antigravity has no key and no program field: the program is the plugin, and the model and the reasoning level are chosen in the row. The type of `agents-acp` has a required `program` field, the executable to run. Every type has the field `parallelRequests`, from 1 to 8 with the default 1, and every agent plugin has the setting `requestTimeout`, the seconds to wait for the program, 120 when it has no value.

A `complete` call runs the program once, with no tool of the vault and no write to disk. Each plugin sets the flags of its program: Claude Code starts with its tools turned off, Codex with a read-only sandbox, and Antigravity without the option that approves every permission request. Codex, Antigravity and `agents-acp` run in an empty temporary folder, and `agents-acp` offers its program no file system, no terminal and no server of tools and refuses every permission request. A program that cannot be found, and a run that outlasts the timeout, are try-next failures (`ReasonUnreachable`, `ReasonTimeout`); any other failed run, and an empty answer, is a stop of unknown reason. The text the program wrote never leaves the plugin.

The plugins run in desktop windows alone (`targets`), so their types are not offered on a phone.
