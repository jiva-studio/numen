# ADR-0025: How this application is tested

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/apps/desktop`, `modules/libs/ui`
- **Related:** ADR-0004, ADR-0006, ADR-0023

## Context

This application is a guest on the person's machine: it walks folders it does not
own, writes into one of them, and keeps a database elsewhere. Every one of those
is something a careless test can do to the machine running it.

It also has one class of bug that is invisible by construction. One database
holds every vault, so a query that forgets its vault returns another vault's
notes without failing, and a test using a single vault cannot see it.

The rules below hold on both sides of the application, in Go and in TypeScript.
Where a rule names a Go helper, the interface side keeps the rule and reaches it
by its own means.

## Decision

### A test never touches the machine it runs on

`t.TempDir` for anything written, `t.Context` for anything cancellable,
`t.Cleanup` for anything opened. No test reads or writes the real configuration,
cache or home directory, and none reaches the network.

### Platform locations are parameters

Where the code defaults to a platform location, a test passes an explicit path.
That is what those paths are parameters for.

### One fixture vault, committed, read-only, awkward

`tests/` holds one vault: a note with no frontmatter, broken YAML, CRLF line
endings, text that is not Latin, a file that is not markdown, and a hidden
folder. Its bytes are the test, so it is excluded from line-ending
normalisation, and a test that needs to change it works on a copy.

Awkward on purpose: a fixture of three tidy notes tests the parser against the
parser's own assumptions, and a fixture of three tidy items tests a component
against its own.

### Generated fixtures where the shape matters and the content does not

The second vault of a scoping test is generated. The awkward parts are awkward
as bytes, and those stay committed.

### A test must be able to fail

For anything guarding a rule rather than a value, the question is not "does it
pass?" but **"would it fail if the rule were broken?"** — and the way to know is
to break it and watch. Remove the filter, invert the condition, delete the
guard, run the test, put it back.

This is required for tests defending something the type system cannot: vault
scoping, invalidation, anything about what is *not* read or *not* returned. It
matters as much on the interface side, where "the nodes were arranged" and "the
component rendered" pass under almost any implementation.

A test that passes either way is worse than no test, because it is counted as
protection and nobody looks at it again.

### A scoping claim is tested with two vaults holding disjoint content

Any test about scoping populates two vaults whose notes share no words, and
asserts both directions: the first vault does not return the second's notes, and
the second does not return the first's. Two vaults pointing at one folder prove
nothing, because a leak then looks like a plausible number.

### CI runs with the race detector, and pooled state with several connections

`go test -race ./...` on every change. Anything configured per connection is
tested by holding several connections at once: a sequential test is handed the
same connection every time, so it cannot observe that the others were never
configured.

### The interface has three levels

1. **The pure core, as plain functions.** No DOM. This is where the assertions
   that matter live, because this is where the decisions are (ADR-0023).
2. **The component in jsdom.** What is emitted, and what is drawn from the props
   it was handed. **The negatives belong here** — what the component does *not*
   emit, what is *not* drawn — because those fail silently and look right in
   every screenshot.
3. **The stories, run as tests.** Each story is rendered in a browser, so a
   story that stops rendering is a failing test, and one fixture serves both the
   gallery and the suite.

## Consequences

- The suite runs on any machine, in any order, without arranging anything first
  and without leaving anything behind.
- Breaking a rule to watch its test fail is a manual step that nothing enforces.
- An awkward fixture is harder to read than a tidy one, and every edge case
  added to it makes every test that counts notes more brittle.
- The stories run in a browser that is not the one the window is built on, so a
  green story is not a drawn window.

## Alternatives considered

**Mocks for the filesystem and the database.** Rejected: the interesting
failures are what SQLite and a real directory tree do — a pragma applied per
connection, FTS5 parsing a hyphen as an operator, a walk following a symlink. A
mock reproduces the model, and it was the model that was wrong.

**A fixture generated in code instead of committed.** Rejected for the vault:
the awkward parts are awkward precisely as bytes — CRLF, a byte order mark, a
name that is not Latin — and generating them in Go hides them from the person
reading the test.
