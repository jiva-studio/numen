# Fetching is a port, and the tools are the machine's

- **Status:** Accepted
- **Date:** 2026-09-06
- **Applies to:** `modules/libs/core`
- **Related:** [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [Where a port is declared and where an adapter stands](0035-where-a-port-is-declared-and-where-an-adapter-stands.md), [A link is a note that carries an address](0038-a-link-is-a-note-that-carries-an-address.md)

## Context

Reaching an address means speaking to a site that changes what it publishes and how, every few weeks. The programs that keep up with that exist, are maintained by people who do nothing else, and are already on the machines of the people who would use this.

## Decision

### The conversation is a port

*What is at this address, and what of it can be had*: what it is called, how long it runs, the words published with it, its sound, a copy of it, and the prose of a page. One port, in the core's own words.

### The tools are run as programs, and named by a setting

`yt-dlp` reaches a video and `ffmpeg` brings sound to what a transcriber opens. Each is named by a setting holding a command and what it is started through, and an empty one asks the path.

**It is a command and not a filename.** A machine that writes the path of a program afresh at every build names whatever does know where it is, and one that keeps several names the one it means.

**Every run is handed the arguments the setting names, before its own.** A site that refuses an unattended request is answered by a flag — cookies from a browser, a token, a runtime that mints one — and which of those a person uses is theirs. What the tool said when it refused is what they are shown: it knows why, and nothing here says it better.

### A machine holding neither tool has no fetcher

Nothing is bound, the ask is answered that this build cannot do it, and the window offers it nowhere from then on. That is what a build without an interface, without a recogniser and without a proofreader already does.

### Neither tool is linked, and neither is shipped

A separate program run over a pipe is neither linking nor distribution, so the system's binary stays the system's. A package that bundled either would take on that program's licence.

## Consequences

- The feature is absent on a machine that has neither tool, and says so.
- What a site does to keep programs out is answered by the person's own
settings, without a release.
- Nothing here is tested against a network: what a run of the real tool says is
recorded, and the tests read that.

## Alternatives considered

**Speaking to the sites from here.** Rejected: it is a full-time job somebody else is already doing, and a month behind it is a feature that does not work.

**Shipping the tools inside the application.** Rejected: it takes on their licences, and it means shipping a program that has to be current to work.
