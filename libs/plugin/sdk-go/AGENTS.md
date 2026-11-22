# libs/plugin/sdk-go

The Go SDK of a native plugin. What it holds, who imports it and when a change is breaking are in [`architecture.md`](../../../.agents/rules/architecture.md) §11 (c) and (j); the packages and their use are in [`README.md`](README.md).

## No wire type in the public API

No exported identifier of a public package names a type of `libs/shared/protocol` or of `connectrpc.com/connect`: not in a signature, a field, a method, a type argument or a constraint. A plugin that needs a message of a contract uses the SDK type of it, and a plugin that needs an error code uses `rpcerror`. Code the SDK packages share and a plugin must not see goes below `internal/`, where the generated clients and the Connect options live (`internal/transport`). `go/sdk-protocol-encapsulation` fails on a leak.

## Exports

- A plugin author reads an exported name as a contract: a rename or a removal is a breaking change (`architecture.md` §11 (j)).
- A constructor takes the SDK's own types. `NewClient` takes an endpoint or a `*http.Client`, never an RPC library type; a constructor that exists only so a test can pass its own transport is unexported.
- A service a plugin serves is mounted with `Mount(s, ...)` on the `server.Server`: `RegisterAuthenticated` for a Connect service, `Register` for a handler that checks a token of its own.
- A host call a plugin makes has one client for one service, in the package of its subject.

## Names

Functions and methods follow the naming rules of [`coding-style-backend.md`](../../../.agents/rules/coding-style-backend.md): imperative verb phrases or predicates; a client method carries the name of the RPC it calls. A test helper package is named `<package>test` and holds what a plugin test needs from that package.

## The protocol version

`server.ProtocolVersion` is `protocol.PluginProtocolVersion` of `libs/shared/protocol`, the same constant `manifest.ProtocolVersion` of the catalog takes. Do not write the number again.
