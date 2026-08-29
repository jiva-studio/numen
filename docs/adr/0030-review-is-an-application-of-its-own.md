# ADR-0030: Review is an application of its own

- **Status:** Accepted
- **Date:** 2026-08-29
- **Applies to:** `modules/apps/desktop` — the binaries; `modules/libs/core` — `usecase/review`
- **Related:** ADR-0002, ADR-0003, ADR-0004, ADR-0008, ADR-0020, ADR-0027, ADR-0028, ADR-0029, ADR-0031

## Context

Cards are written occasionally and run daily. Writing one is an act on a file, in a window built for acting on files; running them is an act over every deck a person has, and it touches no file the person would name.

Where that second activity lives decides what a person has to start in order to do it, and what has to be open while they do.

## Decision

### Review is a binary of its own, beside the editor

`numen-review` stands beside `numen` and `numen-cli`, over the one core. The three ship together, one version, one installer: a review with no editor has nothing to review, so there is no sense in shipping them apart, and ADR-0007's refusal of an index at a version a build does not carry costs nothing between binaries that are always the same build.

It opens on **the vaults and what is due in each**. That is the whole of its front door: what a person owes today, and the place to come back to between decks.

### It reads the registry and never writes it

Which vaults exist, where they are and which was opened last is what the registry answers, and it answers before any database is opened (ADR-0003).

Every write to it rewrites it whole and nothing locks it between processes, so a review recording that it had been started could erase what the editor had just written. It is read here and never written.

### It reads the index and never writes it

The index answers one question for this application: **which files in a vault are decks**. It cannot answer another, because a card's heading reaches it without the mark the card is known by (ADR-0029) — the cards themselves are read from the deck files, whatever else is open.

Nothing is written, so the single write lock ADR-0002 keeps over all vaults is untouched, and ADR-0008's guarantee that writes queue inside one process is never asked to hold across two. A vault the index does not carry is a vault this application says has not been read yet, and it says which application reads it.

### It writes marks into decks, and works from what it reads back

A card typed by hand carries no mark until the application next writes its file (ADR-0028). A person who opens only this application never writes that file in the editor, so their deck would hold cards nothing can be recorded against.

A deck holding a card with no mark is therefore written once, which mints a mark for every card in it, and **the deck is read again afterwards and the session works from that read**.

**It is done when a person sits down to a vault, and never for the counting.** The front door counts every vault the installation holds, and a count that wrote would write into all of them at every launch — including the ones nobody opened. A card with no mark is counted as nothing until the vault it is in is sat down to.

The vault's write lock lives in the process (ADR-0020), and two processes on one vault hold a lock each. A mark minted here while the editor is saving the same deck is a write one of the two loses, and the editor's own save mints marks of its own for whatever is missing them. Reading the deck back is what makes that harmless: a card is reviewed under the mark the file holds, so a stamp that did not land is a card left out of this session and stamped again at the next, and no answer is ever recorded against a mark that is not in the file.

### It hands one thing to the editor

A card written badly is the one thing this application does not fix. It hands the deck's path and the card's mark to the editor, which comes forward with that deck open at that card. Between two processes that is an invocation and not a call.

## Consequences

- **A third binary is built, packaged and released.** Both link against the system's own webview, so both are built on the machine they are built for, and the installer carries two windows.
- **A person can run their cards without knowing an editor exists.** That is the point of the front door, and it is why marks are minted here.
- **A vault that was never scanned cannot be reviewed** until the editor has been opened on it once. The list of decks is the index's answer, and nothing here builds it.
- **Two processes write to one vault.** The write lock does not reach across them. The loss it admits is a mark, never an answer, and a lost mark is minted again at the next session.
- **The editor's index goes stale when a mark is minted here.** The watcher lives in the editor's process, so the deck is read again the next time that process runs. Nothing in the index is wrong meanwhile — a card's heading does not change when it is given a mark.
- **Two windows are open on one vault when a person moves between them**, and each holds its own lock and its own read of the same files.

## Alternatives considered

**A tab in the editor.** Rejected: every tab there is a thing in the vault — a note, a deck, a stencil, a plex, a conversation — and a review is not a thing in the vault. It is an activity over many of them, and the metaphor does not hold.

**A second window of the editor.** Rejected: reaching the daily act through the application built for the occasional one is backwards. Writing cards is occasional; running them is daily.

**Opening the index for writing, so the review keeps it current after minting a mark.** Rejected: it buys a heading that did not change, and it pays with a second writer over the one database ADR-0002 keeps for every vault — an answer queuing behind a scan of a vault it has nothing to do with, and a write coming back busy where the code above it expects to wait.

**Reaching the vault without the index at all**, by walking it for decks at every launch. Rejected: it is the scanner written twice, and the second copy has no watcher, so it is the scanner written twice and worse.

**Refusing to review a card with no mark**, and leaving marks to the editor. Rejected: it makes the editor a prerequisite for the daily act, which is the arrangement this decision exists to avoid.
