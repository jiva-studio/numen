# The schema — what holds here

The conventions every language in this repository shares are in `AGENTS.md` at
the root. This file is the protocol's own, and it holds over
`modules/libs/protocol`.

The decision behind the shape is
[`docs/adr/0034-one-service-to-a-subject.md`](../../../docs/adr/0034-one-service-to-a-subject.md),
and how a client comes from it is
[`docs/adr/0005-a-client-is-generated-from-the-protocol.md`](../../../docs/adr/0005-a-client-is-generated-from-the-protocol.md).

## Generated code is never edited

`*.pb.go` and `*_pb.ts` come from `buf generate`. A change goes into the
`.proto` and is generated again; `make generate-check` fails on a tree whose
generated code is out of date with the schema it came from.

A window's own type that mirrors a message is written by hand and mapped field
by field. Where a window hands a structural type straight to the client, the
two line up by their field names alone and a rename breaks the match with
nothing said.

## A service is one subject

A subject is what a client asks about, named in one word. A question about a
subject is asked of no other service, and a binary mounts a service whole: what
is on a service decides what a binary must be able to answer.

A file is a place and not a subject. Two things that stand beside one file are
two subjects.

## Names

A method is a verb and a noun — `GetSettings`, `WriteNote`, `ListReviewDays`.
A reference field is named after the message it points at. An enum's zero value
either means nothing or is one of the meanings, and which it is is written down.

## Totality

An enum added to the schema does not compile until every place that answers it
is told what to do with the new value. `internal/testsupport/totality.go` walks
an enum's own descriptor and offers `Handled`, `Produced` and `RoundTrip`; a
hand-written union keyed against a generated enum is what this replaces.

## A list that grows with time is paged

A list bounded by the person's own disk over an in-process transport needs no
paging. One that grows with time — a day added for every day reviewed — does.
