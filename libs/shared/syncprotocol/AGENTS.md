# libs/shared/syncprotocol

The protocol between Numen Sync and the sync server, shared by the plugin and by `numen-cloud`. Where a shared library goes is the skill [`nm-place-library`](../../../.agents/skills/nm-place-library/SKILL.md) and `architecture.md` §11 (e).

## A leaf

The module requires `connectrpc.com/connect`, `google.golang.org/protobuf`, `golang.org/x/crypto` and `github.com/klauspost/compress` (for `blobcipher`) and `libs/shared/identity`, which only `fakeserver` imports to check a sign-in as the real server does. Nothing else imports it: a chain and a key are `bytes` in the messages, so a client needs no key code. The `go/shared-leaf` check fails on a requirement of a module outside `libs/shared`.

## Generated code is never edited

`gen/` comes from `buf generate` over `proto/`, and `make generate-check` fails on a tree whose generated code is out of date with the schema. The Go and Connect-Go plugins only: no TypeScript has a consumer.

## One version, no compatibility layer

`Version` is the one protocol version this code implements. Every request carries it in `Numen-Sync-Protocol`, set by `NewVersionInterceptor` on the client and checked by `NewVersionHandlerInterceptor` on the server, so a service added later gets it with no code. A mismatch is `FAILED_PRECONDITION` with an `UpdateRequired` detail and `ErrUpdateRequired` on the client. A change to a message that an older client reads wrongly raises `Version`; nothing negotiates and nothing converts.

## What is signed

`GetSignInMessage` is the frozen byte layout of a sign-in. The core signs it under `numen/v1/plugin-sign/<plugin id>` (`identity.SignForPlugin`), and the server builds the same bytes from its own record of the challenge. A change to the layout is a change of `Version`. `GetChainReadMessage` is the same layout under the label `numen-sync/chain-read/v1`, signed by the recovery key with plain Ed25519 to read the chain of an account. `CheckChallenge` is what a client runs before it signs: the audience is the address it dialed and the expiry is near its own clock.

`GetRevisionMessage` and `GetCatalogEntryMessage` are the frozen layouts of a revision and of a catalog entry, signed the same way under the plugin domain. Both start with a label (`numen-sync/revision/v1`, `numen-sync/catalog/v1`) and a zero byte. Variable fields are a uvarint length and bytes, so a byte moved between two neighbouring fields changes the message; `enc_meta` and `enc_body` enter as their SHA-256; numbers are big endian. The signature and the `seq` are never signed. `GetRevisionID` and `GetCatalogEntryID` are the SHA-256 of the message: the `parent_rev` of the next revision and the `prev_hash` of the next entry. A change to any layout is a change of `Version`. `CheckRevision` and `CheckCatalogEntry` check shape and sizes only, for the client and the server alike; `limits.go` holds the sizes and page limits.

`GetRestoreEpochMessage` is the frozen layout the server signs to announce a restore: the label `numen-sync/restore-epoch/v1`, a zero byte, the 16 bytes of id and `issued_at` as 8 bytes big endian. `CheckRestoreEpoch` checks sizes and the Ed25519 signature against the key `GetRestoreEpoch` returns; `client.RestoreClient` runs it before it returns an epoch. A server that was restored from a backup cannot know it was, so the operator issues the epoch; every `GetChangesResponse` names the current id in `restore_epoch_id`, empty before the first. `GetBlobUrls` answers a blob whose body the store lost with `NOT_FOUND` and a `BlobMissing` detail, read as `BlobMissingError` (it reads as `ErrBlobMissing`, not as `ErrNoVault`). `EncodeCatalogEntry` and `DecodeCatalogEntry` give the bytes a device keeps a copy of a catalog in.

## Clients

The package `client` holds them, so the root package is values only and a use case may import it (`ts/plugins` allows the root, `blobcipher` and `gen`, not `client`). `client.NewCatalogClient`, `NewRevisionClient`, `NewBlobClient` and `NewAccountClient` apply the version interceptor and `NewBearerInterceptor`, which reads the session token again for each request. The refusals of the server are mapped to `ErrWriteNotAllowed` (permission denied), `ErrNoVault`, `ErrVaultExists`, `QuotaExceededError` and `BlobNotReadyError` (from the details), `ErrUnauthenticated`, `ErrUnreachable` and `ErrUpdateRequired`; any other refusal is returned as the server sent it. `client.PutBlob` and `client.GetBlob` move a body over a presigned URL with plain HTTP: no bearer and no version header.

## Multipart and the metadata-only revision

A body above `MaxBlobSize` goes in parts: `ReserveMultipartBlob` (checked by `CheckReserveMultipart`), `GetPartUrls`, `PutPart` of each part, `ListParts` to go on after a break, `CompleteMultipartBlob` (the server builds the part list; the client never sees an ETag) and `AbortMultipartBlob`. `GetBlobRange` reads a stored body from a byte. `blobcipher.Sealer` gives the stored bytes at any offset, so a part is derived again, never kept. A revision may name the blob of the head of its own file with the same hash and size, and the head as parent: that is a move or a rename without a new body. A tombstone has no blob and may have an empty `enc_meta`. `GetChangesResponse.reset` says the cursor is older than the deletion records the server keeps.
