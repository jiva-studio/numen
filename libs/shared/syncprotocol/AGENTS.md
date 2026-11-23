# libs/shared/syncprotocol

The protocol between Numen Sync and the sync server, shared by the plugin and by `numen-cloud`. Where a shared library goes is the skill [`nm-place-library`](../../../.agents/skills/nm-place-library/SKILL.md) and `architecture.md` §11 (e).

## A leaf

The module requires `connectrpc.com/connect`, `google.golang.org/protobuf`, `golang.org/x/crypto` and `github.com/klauspost/compress` (for `blobcipher`) and `libs/shared/identity`, which only `fakeserver` imports to check a sign-in as the real server does. Nothing else imports it: a chain and a key are `bytes` in the messages, so a client needs no key code. The `go/shared-leaf` check fails on a requirement of a module outside `libs/shared`.

## Generated code is never edited

`gen/` comes from `buf generate` over `proto/`, and `make generate-check` fails on a tree whose generated code is out of date with the schema. The Go and Connect-Go plugins only: no TypeScript has a consumer.

## One version, no compatibility layer

`Version` is the one protocol version this code implements. Every request carries it in `Numen-Sync-Protocol`, set by `NewVersionInterceptor` on the client and checked by `NewVersionHandlerInterceptor` on the server, so a service added later gets it with no code. A mismatch is `FAILED_PRECONDITION` with an `UpdateRequired` detail and `ErrUpdateRequired` on the client. A change to a message that an older client reads wrongly raises `Version`; nothing negotiates and nothing converts.

## What is signed

`GetSignInMessage` is the frozen byte layout of a sign-in. The core signs it under `numen/v1/plugin-sign/<plugin id>` (`identity.SignForPlugin`), and the server builds the same bytes from its own record of the challenge. A change to the layout is a change of `Version`. `GetChainReadMessage` is the same layout under the label `numen-sync/chain-read/v1`, signed by the recovery key under `numen/v1/recovery-sign/<plugin id>` and a zero byte (`identity.SignForRecovery`, `SignAsRecovery`; the server checks it with `identity.VerifyForRecovery` under `SignInPluginID`), not as plain Ed25519 over the message, to read the chain of an account. `CheckChallenge` is what a client runs before it signs: the audience is the address it dialed and the expiry is near its own clock.

`GetRevisionMessage` and `GetCatalogEntryMessage` are the frozen layouts of a revision and of a catalog entry, signed the same way under the plugin domain. Both start with a label (`numen-sync/revision/v1`, `numen-sync/catalog/v1`) and a zero byte. Variable fields are a uvarint length and bytes, so a byte moved between two neighbouring fields changes the message; `enc_meta` and `enc_body` enter as their SHA-256; numbers are big endian. The signature and the `seq` are never signed. `GetRevisionID` and `GetCatalogEntryID` are the SHA-256 of the message: the `parent_rev` of the next revision and the `prev_hash` of the next entry. A change to any layout is a change of `Version`. `CheckRevision` and `CheckCatalogEntry` check shape and sizes only, for the client and the server alike; `limits.go` holds the sizes and page limits.

## Clients

`NewCatalogClient`, `NewRevisionClient`, `NewBlobClient` and `NewAccountClient` apply the version interceptor and `NewBearerInterceptor`, which reads the session token again for each request. The refusals of the server are mapped to `ErrWriteNotAllowed` (permission denied), `ErrNoVault`, `ErrVaultExists`, `QuotaExceededError` and `BlobNotReadyError` (from the details), `ErrUnauthenticated`, `ErrUnreachable` and `ErrUpdateRequired`; any other refusal is returned as the server sent it. `PutBlob` and `GetBlob` move a body over a presigned URL with plain HTTP: no bearer and no version header.
