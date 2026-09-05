# An answer is an artifact, a schedule is a cache

- **Status:** Accepted
- **Date:** 2026-08-29
- **Applies to:** the vault format, and `modules/libs/core` — `review`, `usecase/flashcards`
- **Related:** [Files on disk are the source of truth](0001-files-are-the-source-of-truth.md), [A vault carries its identity, and application state lives with the application](0003-a-vault-carries-its-identity.md), [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md), [One process, one lifetime](0020-one-process-one-lifetime.md), [The stencil, the deck and the card](0027-the-stencil-and-the-deck.md), [Review is an application of its own](0030-review-is-an-application-of-its-own.md)

## Context

A person writes their own notes. They do not write their answers: a year of them is the record of an hour a day, and nothing on any machine can produce it a second time. It is the one thing in this system that is irreplaceable, and it is also the thing every scheduler treats as a by-product of the state it keeps.

Anki keeps the state — due, ease, interval — and changes it on every answer. Two things follow. Changing the algorithm means converting state that has no honest conversion, because SM-2's ease is not FSRS's difficulty and pretending otherwise quietly spoils everybody's schedule. And carrying a vault between two machines means merging state that both of them changed.

The two kinds of thing are already separate: what a machine can work out again is a cache, and what it cannot is an artifact that lives in the vault.

## Decision

### The answer is kept and the schedule is worked out

**An answer is recorded. A schedule is computed from the answers and thrown away freely.**

An answer says which card, which face, when, how well it went, and how long the card stood on the screen. It says nothing about intervals, ease or difficulty: everything a scheduler works out is worked out again, so changing the scheduler replays the history rather than converting a state that cannot be converted.

### The log lives in the vault, one file to a run

```
<vault>/.numen/flashcards/<ulid>.jsonl
```

A run of review opens one file, appends to it while it lasts, and never touches it again. The name is a ULID, so the files sort in the order they were made.

**One writer to a file is what makes merging concatenation.** A file synchroniser has no way to merge two appends to one file: it carries a file whole, and two machines that both appended leave it a conflict to resolve or a version to lose. When each run writes only its own file, the whole history is the union of the files, and no two machines ever write one of them. A phone and a desktop synchronising one vault come away with one history and one set of counts.

One line is one answer:

```json
{"v":1,"id":"01K3ZQ7X2M9QRSTVWXYZ012345","card":"k7m2xq9fzp","face":"Recognise","at":"2026-08-29T09:12:33.412Z","rating":3,"ms":4210}
```

and one line takes an answer back:

```json
{"v":1,"id":"01K3ZQ7X8B0CDEFGHJKMNPQRST","undo":"01K3ZQ7X2M9QRSTVWXYZ012345","at":"2026-08-29T09:12:41.006Z"}
```

An answer carries an identifier of its own so that taking it back names it exactly. Replay skips an answer some line takes back, and the two lines both stay in the file: **nothing here is ever rewritten or removed.**

**One identifier is one answer**, however many lines carry it. A synchroniser that met a conflict leaves a second copy of a run beside the first, and a person restoring a backup puts one there by hand; counting those lines twice would double what a card has been through and send it away for longer than it was earned.

A line that does not parse is skipped and counted, and so is a line whose `v` is a version this build does not know. A run that ended when the machine did leaves a torn last line, which is one of those: an append is not atomic, and the file has to be readable anyway.

### A schedule belongs to a card and one of its faces

**A card face is what carries a schedule**: one card, and one face of the stencil that cuts it. A card is shown once through each face its stencil declares, and each face asks a different thing of the person, so each face has a path of its own.

The card half is the mark it is known by, which travels with it: a card moved to another deck or another vault keeps its schedule and its history, and nothing is recomputed.

The face half is the face's name, which is its heading in the stencil, and a face carries no mark. **Renaming a face starts its schedule again.** That is the cost of this decision, and it is paid where it falls: a stencil is written once and used a thousand times, and a face is renamed about as often as it is written.

### The cache lives with the application

```
<cache dir>/numen/flashcards/<vault id>.json
```

The service folder inside a vault holds what the application made and cannot make again. A schedule is made again by definition — that is what makes the answers an artifact and the schedule not one. It is also the one file here that is rewritten whole, and a file rewritten whole inside a synchronised folder is a file that conflicts.

So it lives where the installation's own state lives, keyed by the vault's identity, and it is deleted at any time at no cost but a replay.

It holds what a launch would otherwise read every answer to learn: the schedule of each card face, the run files it was worked out from with the length each of them had, and the name of the scheduler that computed it, parameters and all. Listing the folder is what a launch does anyway; the cache is what saves it opening the files.

### A run the cache does not name is the history read again

A schedule depends on the order of the answers, so an answer arriving after later ones have already been counted cannot be added to what they produced.

Synchronisation delivers exactly that: last night's run from the phone lands after this morning's run on the desktop has been counted. **So nothing is ever added to a schedule after the fact.** The cache records the runs it was worked out from, each with the length it had, and a folder differing from that in any way is a history read from the beginning — which is also what a cache filled by another scheduler, or by another shape of this file, comes to.

**The length is what says a run has changed.** A sitting appends to one file all evening under one name, so a cache going by names alone would call itself current from the first answer of that sitting and never count the rest of it.

### The scheduler is a port, and the cache says which one filled it

Review is graded, not remembered-or-not: **again, hard, good, easy**. How well a thing was recalled is what a scheduler has to be told to space anything sensibly.

Behind the port stands FSRS. The cache records the scheduler's name and the parameters it was running on, because the weights are what the numbers mean: stability and difficulty read under other weights are a wrong day given confidently. A cache filled by another name is thrown away whole and computed again from the log — which is what the log is for.

### An answer whose card is gone is kept

A mark in the log that no deck holds is not an error and is never removed. The card may have been moved to a vault that has not been synchronised yet, or taken out and put back. If it returns, its mark returns with it, and the history is its own again.

## Consequences

- **A vault carries its own history.** Copied to another machine, its cards are the cards a person has been answering, at the interval they had reached. This is what [The stencil, the deck and the card](0027-the-stencil-and-the-deck.md) recorded as the cost of keeping the schedule outside the vault, and it is the cost this decision takes back — the deck file still holds no schedule and is still not rewritten when a card is answered.
- **The order of the history is the order of the clocks that wrote it.** Two machines whose clocks disagree interleave their answers wrongly, and nothing here can tell.
- **The mark the schedule cache is filed under carries the placement.** A preset's shares, whether its load is evened, the goal that decides whether it is evened at all, and the hour a day begins at all decide which day a card lands on, so all of them stand in the mark.
- **The log grows, one small file to a run.** Daily review is on the order of a few hundred files a year, each a few kilobytes. Folding old ones together is a rewrite, and a rewrite is the thing that makes merging hard, so it is only ever done to months nothing writes to any more.
- **A store of derived files has to be able to say what names it holds.** Reading the log means reading every file in one folder, which `DerivedStore` had no way to ask.
- **A stencil's faces are renamed knowingly.** The editor is where a person is told what a rename costs.
- **A launch that meets a run it has not counted reads the whole history.** That is every launch a person answered anything at, and it is a few hundred kilobytes of text a year.

## Alternatives considered

**One log file for the vault.** Rejected: two machines appending to one file is the case no file synchroniser handles. The best of them leaves a conflicted copy beside it and the worst keeps one version, and what is lost is the one thing that cannot be made again.

**One file per device per month.** Rejected: the device is a stand-in for one writer, and the run already is one. Naming files after the device means remembering an identity for the installation, in a file somebody has to keep, for no gain over naming them after the run that wrote them.

**One file per card.** Rejected: fifty thousand cards is fifty thousand files, which is bad for a filesystem and worse for anything that synchronises one.

**The schedule beside the card in the deck**, as org-fc and the Obsidian plugin do. Rejected: the file is rewritten after every answer, so a session is a diff over the whole deck, and what the person wrote sits in one file with what a machine counted, under one modification time.

**The schedule in the index.** Rejected: it does not travel with the vault, which is the whole of what this decision is for, and it makes the review a writer over the one database every vault shares.

**State kept and mutated, as Anki does.** Rejected for the two reasons in the context. The decisive one is that changing the algorithm is a certainty over the life of this product and not a hypothesis.

**A schedule for the card rather than for each of its faces.** Rejected: the faces exist because they ask different things, and one schedule over all of them either shows a person what they know or hides what they do not.

**Giving each face a mark of its own**, so a rename costs nothing. Rejected for now: it writes a machine's token into the stencil beside every face, which is the shape turned down for cards until there was a reason. A rename costing a schedule is the reason, when it turns out to be one.
