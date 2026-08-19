# ADR-0035: What the application cannot do, it says

- **Status:** Accepted
- **Date:** 2026-08-19
- **Applies to:** `modules/apps/desktop`
- **Partly supersedes:** ADR-0015 — what happens to an index this build cannot account for
- **Related:** ADR-0000, ADR-0002, ADR-0015, ADR-0033

## Context

An application starts by putting several things together: settings, a list of
vaults, an index, an agent, the interface itself. Any of them can be missing or
broken on somebody's machine.

Three answers are available for each, and only one of them is usually right:

- **carry on without it**, when the thing is not what the application is for;
- **say so and stop**, when it is;
- **repair it**, which for a derived store means deleting and building again.

The third is the one that looks free and is not. The index is a cache by
ADR-0000's rule and the vaults are the truth, so throwing it away and reading
the vaults again is always correct — except that it is not free: it costs
minutes of reading, and it costs whatever the index held that was bought rather
than derived (ADR-0000, the third class). An application that repairs by
deleting spends a person's money and time on its own initiative, to recover from
a state it did not diagnose.

The second is the one that is easy to do badly. A message written to standard
error is a message nobody sees: an application opened from a dock or a desktop
entry has no terminal, and refusing there is indistinguishable from crashing.

## Decision

**What the application cannot do, it says — where the person is, and with what
it knows.**

- **Refusal is for what stops the work, and nothing else.** Two states stop it:
  settings that cannot be read, and an index that cannot be opened. Everything
  else the application starts with is something it can do without and name: a
  vault it does not have, an agent nobody can reach, a model that did not
  answer.

- **A refusal is drawn in a window.** The application opens one and puts in it
  what stopped it, the facts it holds about the state it found — which file,
  which numbers, which build — and what to do about it. It also writes one line
  to standard error, for a terminal and for a log.

- **Nothing is repaired by deleting.** A state this build cannot account for is
  reported and left exactly as it stands. Only a numbered migration empties
  anything, about its own tables, saying so in its own file (ADR-0015).

- **An index at a version this build does not carry is refused.** A schema
  written by a later build holds what this one cannot read; carrying on, this
  build would write to it by rules that no longer apply and damage it. The
  remedy named is the build that wrote it.

- **A refusal names no remedy that costs the person something they did not ask
  to spend.** "Update numen" is a remedy. "Delete your index" is a bill, and it
  is not this application's to write.

- **An installation with no vault is given one.** A folder is made where the
  system says this person keeps documents, and it is theirs: ordinary files in
  an ordinary place, which they may move or replace. An application that answers
  "you have no vault" and stops asks somebody to read a manual before it will do
  anything at all.

## Consequences

**Positive**

- The state "the application did not start and said nothing" stops existing.
- A person who has just installed the application can write something in it.
- An index is never spent to recover from a state nobody diagnosed, which is
  what makes ADR-0000's third class survivable in practice.
- A refusal carries the two numbers, the path and the revision, so a report
  about one is a report and not a guess.

**Negative / costs**

- The refusal window is a second interface, small and separate from the one the
  application serves. It has to keep working when the first one cannot be built,
  so it carries its own markup and its own styling, and nothing shared can be
  reached from it.
- A first vault made without being asked for is a folder somebody did not
  create. It is empty, it is named, and it is where they would have put one.
- Two states are now fatal by decision rather than by accident, and adding a
  third is a change to this record rather than a line of code.

## Alternatives considered

**Carry on without the index.** The notes are files and can be read and written
without it, so a degraded mode is possible: no search, no plex, no backlinks.
Rejected. The index is what every question the window asks goes through, so each
one would need an answer meaning "not now", and the application would be
half-working for an unbounded time without saying for how long. Where the index
is fine and the build is old, it is also the wrong answer: the person should
update, not work at half strength.

**Refuse on standard error, as before.** Rejected: it is the same as crashing
for everyone who does not start the application from a terminal, which is
everyone using it as an application.

**Repair by emptying, as the index once did on a schema it could not account
for.** Rejected. It is the behaviour this decision exists to name: it destroyed
what an installation had paid for, in response to a state it had not diagnosed,
without asking.
