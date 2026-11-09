# Protocol Schemas and Contracts

The conventions shared across all languages in this repository are in `AGENTS.md` at
the root. This file defines specific rules for `libs/core/protocol`. Where wire types and clients belong, and which library a client goes to, is the skill [`nm-place-library`](../../../.agents/skills/nm-place-library/SKILL.md).

Protocol architecture requires one service per subject domain, with clients
generated automatically from `.proto` definitions.

## Generated code is never edited

`*.pb.go` and `*_pb.ts` come from `buf generate`. A change is made in the
`.proto`, and the code is generated again; `make generate-check` fails on a tree whose
generated code is out of date with the schema it came from.

A window's own type that mirrors a message is written by hand and mapped field
by field. Where a window passes a structural type directly to the client, the
two match by their field names alone and a rename breaks the match.

## A service is one subject

A subject is what a client queries, named in one word. A query about a
subject belongs to that single service, and a binary mounts a service whole.

A file path is a storage location, not a domain subject. Two separate concepts
associated with the same file remain two distinct subjects.

## An error code is a successful response; a status code is not

A call that cannot be served at all — nothing mounts it, the transport is
shutting down — fails with a status code. A call that was served and returned a negative result —
a name already taken, a vault not in the list — sets the `ErrorCode`
field on the response message instead, and the result field stays unset. A client reading the status to tell the two apart is
reading the wrong signal. This is a convention expressed in the shape of each
response message; no lint checks it yet.

## Names

A method is a verb and a noun — `GetSettings`, `WriteNote`, `ListReviewDays`.
A reference field is named after the message it points at. An enum's zero value
either means nothing or is one of the meanings, and which it is is written down.

## Totality

An enum added to the schema does not compile until every place that handles it
has a case for the new value. `internal/testsupport/totality.go` iterates over
an enum's own descriptor and provides `Handled`, `Produced` and `RoundTrip`; a
hand-written union keyed against a generated enum is what this replaces.

## A list that grows with time is paged

A list bounded by the person's own disk over an in-process transport needs no
paging. One that grows with time — a day added for every day reviewed — does.
