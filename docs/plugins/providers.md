# Writing a provider

A provider is a plugin that carries out operations for the connections of the types it declares. The core does the rest: it keeps the connections and their keys, reads the rows of the plugin that asks, tries them in order, and tells the person when one is set aside. A provider holds no state and no key between calls.

## Declare the types

The manifest lists the types under `contributes.connectionTypes` (see [the manifest fields](README.md#manifest-fields)), declares `numen.v1.ProviderService` under `services`, and has a `backend`. `operations` names what a type carries out: `complete` and `embed` today. A type lists only the operations its provider fills with a role.

## Fill the roles

`plugin-sdk-go/connection` has one interface for each role, and a `Provider` struct that lists the roles a plugin fills. `connection.Mount(s, provider)` serves them on the plugin server; a role left nil answers as not served.

| Role            | Methods                                       | Operation                          |
| --------------- | --------------------------------------------- | ---------------------------------- |
| `Completer`     | `Complete`                                    | `complete`                         |
| `Embedder`      | `Embed`                                       | `embed`                            |
| `ModelLister`   | `ListModels`, `ListReasoningLevels`           | the editor lists models and levels |
| `ProfileReader` | `GetModelProfile`, `ProbeEmbeddingWidth`      | `embed`                            |

Every method receives a `Call`: the type, the fields that are not secret with the fixed values of the type, the secret fields, the model and the reasoning level. The core reads the secrets once for the call, from a snapshot of the connection, and refuses the call if the connection was edited meanwhile. `Secret` prints, logs and encodes without its value; read it from the `Value` field, send it to the service and keep it nowhere. The model is the name the person typed or chose: do not check it against the list of the service. `Embed` returns raw vectors, and the plugin that asks applies the prefixes of the profile.

`GetModelProfile` returns `IsKnown: false` for a model it has no profile for, and the core then calls `ProbeEmbeddingWidth`, which embeds one short text and returns the width. `MaxTokens` is 0 when the limit is not known.

The calls `SubmitJob`, `GetJobStatus` and `FetchJobResult` are in the schema for deferred requests. The core does not call them, and the SDK does not serve them.

## Report failures

A call returns a result, one of two failures, or the error of its context. A failure carries a reason from a fixed list, and the core never sees an HTTP code or a text from the service.

| Failure                        | Meaning                                              | Constructor                                |
| ------------------------------ | ---------------------------------------------------- | ------------------------------------------ |
| try the next                   | this connection cannot serve now                     | `NewTryNextError(reason, retryAfter)`      |
| stop                           | the request itself is wrong                          | `NewStopError(reason)`                     |

A try-next failure is a rejected key (401, 403: `ReasonKeyRejected`), a payment required (402: `ReasonPaymentRequired`), a rate limit (429: `ReasonRateLimited`, with its Retry-After as `retryAfter`), a service that cannot be reached or answers 502, 503 or 504 (`ReasonUnreachable`) and a timeout (`ReasonTimeout`). A stop is a missing model (404) or any other 4xx (`ReasonBadRequest`) and a 500 (`ReasonUnknown`). An error that is neither is a stop of unknown reason, and its text does not leave the plugin. A cancelled call returns `ctx.Err()`, which travels as the status code `Canceled`: the outbound request of the provider carries the same `ctx`, so a cancelled call ends the request to the service.

The core retries a timeout or an unreachable service once. After that it sets the connection aside for 60 seconds, or for the Retry-After the service gave, at most 10 minutes. After a rejected key or a payment required it stays aside until the connection is edited. The window tells the person when a connection was set aside and the next row answered, once for each connection and reason. When every row fails, the plugin that asked gets an error that names each connection and its reason.

A provider checks the address it resolves to before it sends a key: the core checks the address as typed and resolves no host name.

## Ask for an operation

A plugin that needs a model does not call a provider. `connection.NewClient`, `NewClientFromEnv` or `NewClientFromVars` returns a `Client` with `Complete`, `Embed` and `GetEmbeddingProfile`. The request names no connection, no model and no key: the core takes the connections from the rows in the settings of the plugin, in the order the person set, and a plugin cannot use a connection that is not in its rows. The errors are `connection.ErrNoConnection` (no row serves the operation), `*connection.AllFailedError` and `*connection.RejectedError`; use `errors.Is` and `errors.As`.

A plugin compiled into the core is called through the in-process pipe, as a plugin process is called over the loopback, and the contract names no transport.
