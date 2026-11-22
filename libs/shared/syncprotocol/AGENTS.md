# libs/shared/syncprotocol

The protocol between Numen Sync and the sync server, shared by the plugin and by `numen-cloud`. Where a shared library goes is the skill [`nm-place-library`](../../../.agents/skills/nm-place-library/SKILL.md) and `architecture.md` §11 (e).

## A leaf

The module requires `connectrpc.com/connect` and `google.golang.org/protobuf` and no module of this repository. It does not import `libs/shared/identity`: a chain and a key are `bytes` in the messages, so a client needs no key code. The `go/shared-leaf` check fails on a requirement of another module.

## Generated code is never edited

`gen/` comes from `buf generate` over `proto/`, and `make generate-check` fails on a tree whose generated code is out of date with the schema. The Go and Connect-Go plugins only: no TypeScript has a consumer.

## One version, no compatibility layer

`Version` is the one protocol version this code speaks. Every request carries it in `Numen-Sync-Protocol`, set by `NewVersionInterceptor` on the client and checked by `NewVersionHandlerInterceptor` on the server, so a service added later gets it with no code. A mismatch is `FAILED_PRECONDITION` with an `UpdateRequired` detail and `ErrUpdateRequired` on the client. A change to a message that an older client reads wrongly raises `Version`; nothing negotiates and nothing converts.

## What is signed

`GetSignInMessage` is the frozen byte layout of a sign-in. The core signs it under `numen/v1/plugin-sign/<plugin id>` (`identity.SignForPlugin`), and the server builds the same bytes from its own record of the challenge. A change to the layout is a change of `Version`. `CheckChallenge` is what a client runs before it signs: the audience is the address it dialled and the expiry is near its own clock.
