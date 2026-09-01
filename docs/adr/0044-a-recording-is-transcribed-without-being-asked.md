# ADR-0044: A recording is transcribed without being asked

- **Status:** Accepted
- **Date:** 2026-09-01
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`
- **Related:** ADR-0008, ADR-0015, ADR-0042, ADR-0043

## Context

ADR-0015 settles that nothing chooses automatically between a document's own text layer and a model's reading of it: there is no reliable test of a text layer, so the layer is used and recognition is something a person asks for.

A recording has no layer to choose against. It says nothing until a model has listened, and no question about the file can change that.

## Decision

### The scan puts recordings in a queue

A recording the index holds no transcript for is work to be done, and the background scan finds it as it finds everything else. One run at a time, and the queue is not stored: it is the answer to a question the index already answers, so a run interrupted by the application closing is picked up when it opens.

An installation may turn this off and ask by hand instead. A vault of a hundred hours is a day of a machine, and how much of it to spend is the person's.

### One heavy run on a machine

Reading a scan and listening to a recording both hold the models and the processor. They take turns.

### Every ending is an answer, and only one of them is "later"

A queue that offers work again forever needs the difference between work not yet done and work that will never come out.

- **Nothing to hear.** Silence and music are answers. They are written down as
answers, and the recording is not offered again.
- **The file will not open.** A container nothing here decodes, or bytes that
are not what they claim. Written down with its reason.
- **Somebody else holds it.** One run to a recording. This one, and only this
one, means come back later.
- **Done.**

A person asks for a recording to be tried again by taking its answer away.

### Changing the model does not re-listen to the vault

What listened is recorded beside what it produced, so a transcript made by something else can be found. Making them again is hours of a machine and is asked for.

## Consequences

- Dropping a recording into a folder is the whole gesture.
- A vault of recordings is a machine at work for as long as it takes, and the
setting that stops it is the only thing that stops it.
- An answer is a file, so a person who deletes the service folder is asking for
every recording in the vault to be heard again.
- Recognition and transcription cannot both be running, so a person who asked
for a scan waits for it before a recording is heard.

## Alternatives considered

**A person asks, as they do for a scan.** Rejected: the reason recognition is asked for is that the alternative is guessing whether a text layer is any good. A recording has nothing to guess about.

**A stored queue.** Rejected: the index already answers which recordings have no transcript, and a second list of the same fact is a second thing to keep true.

**Retrying a failure after a while.** Rejected: bytes that will not decode will not decode later, and a queue that comes back to them spends a machine on the same answer.
