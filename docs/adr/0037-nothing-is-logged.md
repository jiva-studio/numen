# Nothing is logged, and a person is told where they are

- **Status:** Accepted
- **Date:** 2026-09-05
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`, `modules/apps/mobile`
- **Related:** [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [Files on disk are the source of truth](0001-files-are-the-source-of-truth.md), [One process, one lifetime](0020-one-process-one-lifetime.md), [One service to a subject](0034-one-service-to-a-subject.md), [How this application is tested](0025-how-this-application-is-tested.md)

## Context

Not one Go package here calls a logging function. There is no `log`, no `log/slog`, no third-party logger, and no file a run appends to. What this application says, it says in four places: the window it is showing, a line on a stream an entry point was handed, the error a call answers with, and `port.Trouble`.

That is not an omission waiting for somebody to bind a logger. It is the shape the destinations already have. What has never been written down is who each destination reaches, and until that is settled, "this should be logged" is a sentence with no object: a machine nobody runs, a file nobody opens, or a person who is looking at a window and would rather be told there.

Two of the four reach a person. The window does, because the person is in front of it. A stream mostly does not: the desktop application ships as a bundle and is started from a launcher, and its standard error goes to whoever started the process — a terminal, if the person opened one, and otherwise nobody. A message a person has to act on that is only on that stream has not been said.

## Decision

### There is no log, and no logging package is imported

`log`, `log/slog` and every logging library are refused in every module here. There is no log file, no log level, no logger handed to anything, and no `*slog.Logger` among a use case's dependencies. → `TestNothingOfTheCoreLogs`, `TestNoApplicationLogs`

A logger is for a volume of messages nobody reads one by one, kept for somebody who was not there when they were made. This is one process, with one lifetime, on the machine of the person using it, showing them two windows. The person is there.

### An answer is not telling

Two different things leave this application, and only one of them is what this record governs.

**An answer** is what somebody asked for: a command's output, a call's return value, and the error a call answers with when it could not. An answer carries whatever was asked about, a person's own text included — `numen-cli search` prints the passages it found, because that is the question.

**Telling** is what nobody asked for: what went wrong in work the application carries on past, and what it is doing behind the window. Everything below is about telling.

### A person is told in the window they are looking at

Three routes, and each is part of what the product shows rather than a convenience for whoever is debugging it:

- `task.Tasks` — work that is running, and, under `Failed`, work that stopped badly and stays in the list until whoever put it there takes it out;
- a `Reason` on the window's own service — `Failed`, `Unwatched`, `Unreachable`: a state the window shows because it is not so;
- the refusal window, for the two states that stop the application from starting at all. See [what the application says when it cannot start](../starting.md).

Anything a person can act on goes to one of these three. A message that exists only on a stream is a message the application decided not to say.

### A stream is written to by what a person started in a terminal

`adapter/cli` writes its answers to the writers it was handed, because a command line's output is the command line. An entry point writes one line to standard error for what stopped it, for a terminal and for whatever collects a process that failed to start; the refusal window says the same thing, drawn.

An application's `cmd/` is the only place naming `os.Stdout` or `os.Stderr`. Everything above it is handed an `io.Writer`, which is what makes any of it testable.

### The core makes no message of its own

Settled in [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md) and not restated here. `port.Trouble` carries an `error` and not a formatted line, and what words it is put into is decided by whoever bound it — which is why the same trouble becomes a `Reason` in a window and a prefixed line in a terminal without the core knowing there are two.

### A message has no level

There are three conditions here, and each already has a place of its own: a call could not answer, and that is its error; work carried on past went wrong, and that is `port.Trouble`; work a person is watching stopped badly, and that is `Failed` on its task. A fourth condition is an entry on this list, not a severity on a message.

Levels are a filter, and a filter is for a reader wading through more than they want. Nobody wades here.

### A message is prose, in the words the window uses

The words a person is told something in are the words the window would use for it: "not watching Notes: the folder went", not `level=warn event=watch_failed`. Structure is for a machine reading, and no machine reads these. A line on a stream names the binary or the subject it belongs to first — `numen:`, `agents:`, `themes:` — because two binaries and several subjects share one terminal.

### A person's own material is answered, never told

Telling never carries a note's title or its prose, a card's text, a search somebody typed, a path under the vault, a transcript's words, or an agent's conversation. The vault's name and a path a person typed on a command line stand: they put them there, and they are reading the line they caused.

Telling goes to a stream whose far end this application does not own — a terminal's scrollback, whatever the launcher keeps, a report somebody sends on. What a person keeps in a vault is theirs, and this application is not the one to leave a second copy of it somewhere they did not choose. [Files on disk are the source of truth](0001-files-are-the-source-of-truth.md): one copy, where they put it.

The window is not held to this. What it shows stands on the screen the material is already on and goes when the window goes, so a task naming the file it is reading is right to.

## Consequences

- **A message is made where the person is, or it is not made.** Where a new one goes is a choice between four places, and there is no fifth to fall back on.
- **Nothing is kept.** A fault nobody was in front of leaves with the process. What answers "what happened" here is a test — see [How this application is tested](0025-how-this-application-is-tested.md).
- **A person filing a bug has the window's own words**, and the lines on their terminal if they started the binary from one. That is what a report is made of.
- **A kind of telling is added as an entry in `task.Tasks` or a `Reason` on a window's service**, which is a change to what the product shows and is reviewed as one.
- **A logging package arriving is refused by a guard**, so binding one is a decision taken in this record rather than in an import block.

## Alternatives considered

**`slog`, bound in `container/` and handed down as a port.** Rejected. It reaches nobody: its default handler writes to standard error, which is the stream the core is already forbidden and which the window's person never reads. A `*slog.Logger` is a concrete type, so a use case taking one holds a library rather than a port, and an interface wrapping it would be `port.Trouble` with levels bolted on. Neither layer guard would have caught it — `log/slog` is not the machine and reaches no disk — which is why this is written down instead of left to the guards.

**A log file beside the index.** Rejected. Nobody opens it, and it is where the privacy fault lands: the telling this application makes is thick with the names of a person's own files, and a file on disk outliving the run is a copy of their material in a place they did not choose. An index can be thrown away and made again; a log holds things no scan would put back.

**Levels on `port.Trouble`.** Rejected. Nothing filters, and a level on a channel with one reader is a field that reader ignores.

**Telling everything on standard error as well as in the window.** Rejected. It is what an entry point does for the few states that happen before a window is up, and doing it for everything makes standard error a log under another name, with the same far end nobody owns.
