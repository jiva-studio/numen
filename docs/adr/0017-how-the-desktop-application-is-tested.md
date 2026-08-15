# ADR-0017: How the desktop application is tested

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0001, ADR-0002, ADR-0014

## Context

This application is a guest on the user's machine: it walks folders it does not
own, writes into one of them, and keeps a database elsewhere. Every one of those
is something a careless test can do to the machine running it.

It also has one class of bug that is invisible by construction. One database
holds every vault, so a query that forgets its vault returns another vault's
notes without failing — and a test that uses a single vault, or two vaults with
the same content, cannot see it. That is not a hypothetical: the first version of
that test passed with the vault filter removed entirely.

## Decision

### A test never touches the machine it runs on

`t.TempDir` for anything written, `t.Context` for anything cancellable,
`t.Cleanup` for anything opened. No test reads or writes the real configuration,
cache or home directory, and none of them reaches the network. Where the code
defaults to a platform location, the test passes an explicit path instead — which
is also why those paths are parameters rather than constants.

### Fixtures live in `tests/` and are read-only

One fixture vault, deliberately awkward: a note with no frontmatter, broken YAML,
CRLF line endings, text that is not Latin, a file that is not markdown, and a
hidden folder. Its bytes are the test, so it is excluded from line-ending
normalisation, and a test that needs to change it works on a copy.

Awkward on purpose: a fixture made of three tidy notes tests the parser against
the parser's own assumptions.

### A test must be able to fail

For anything that guards a rule rather than a value, the question is not "does it
pass?" but **"would it fail if the rule were broken?"** — and the way to know is
to break it and watch. Remove the filter, invert the condition, delete the guard,
run the test, put it back.

This is required, not encouraged, for tests that defend something the type system
cannot: vault scoping, invalidation, anything about what is *not* read or *not*
returned. A test that passes either way is worse than no test, because it is
counted as protection and nobody looks at it again.

### Two vaults, with different content

Any test about scoping populates two vaults whose notes share no words, and
asserts both directions: the first vault does not return the second's notes, and
the second does not return the first's. Two vaults pointing at the same folder
prove nothing, because a leak then looks like a plausible number.

### Concurrency is tested with `-race`, and pooled state with several connections

CI runs the suite under the race detector. Anything configured per connection is
tested by holding several connections at once: a sequential test is handed the
same connection every time, so it cannot observe that the others were never
configured.

## Consequences

**Positive**

- The suite can be run on any machine, in any order, without arranging anything
  first and without leaving anything behind.
- The tests that matter most are the ones proven to fail when they should.

**Negative**

- Breaking a rule to watch its test fail is a manual step that nothing enforces.
  It is written down because it is a habit rather than a mechanism, and a habit
  that is not written down is not a habit.
- An awkward fixture is harder to read than a tidy one, and every new edge case
  added to it makes every test that counts notes slightly more brittle.

## Alternatives considered

**Mocks for the filesystem and the database.** Rejected: the interesting failures
here are what SQLite and a real directory tree actually do — a pragma applied per
connection, FTS5 parsing a hyphen as an operator, a walk following a symlink. A
mock reproduces the model, and it was the model that was wrong.

**A fixture generated in code instead of committed.** Rejected for the vault: the
awkward parts are awkward precisely as bytes — CRLF, a byte order mark, a name
that is not Latin — and generating them in Go hides them from the person reading
the test. Generated fixtures are used where the content is uninteresting and the
shape matters, as in the second vault of a scoping test.
