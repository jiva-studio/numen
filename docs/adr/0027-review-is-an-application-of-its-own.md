# Review is an application of its own

- **Status:** Accepted
- **Date:** 2026-08-29
- **Applies to:** `modules/apps/desktop` — the binaries; `modules/libs/core`
- **Related:** [One database for all vaults, outside them](0002-one-database-for-all-vaults.md), [A vault carries its identity, and application state lives with the application](0003-a-vault-carries-its-identity.md), [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [A vault is scanned in the background](0008-a-vault-is-scanned-in-the-background.md), [One process, one lifetime](0020-one-process-one-lifetime.md), [An agent reaches the vault through tools](0021-an-agent-reaches-the-vault-through-tools.md), [The agent this application starts is a port](0022-the-agent-this-application-starts-is-a-port.md), [How this application is tested](0025-how-this-application-is-tested.md), [The stencil, the deck and the card](0026-the-stencil-and-the-deck.md), [An answer is an artifact, a schedule is a cache](0028-an-answer-is-an-artifact-a-schedule-is-a-cache.md), [Both windows open a vault through one path](0033-both-windows-open-a-vault-through-one-path.md)

## Context

Cards are written occasionally and run daily. Writing one is an act on a file, in a window built for acting on files; running them is an act over every deck a person has, and it touches no file the person would name.

Where that second activity lives decides what a person has to start in order to do it, and what has to be open while they do.

## Decision

### Review is a binary of its own, beside the editor

It is called **Flashcards**. That is the thing a person has and wants to run; review is what is done to it, and every application reviews something.

`numen-flashcards` stands beside `numen` and `numen-cli`, over the one core. The three ship together, one version, one installer: flashcards with no editor has nothing to run, so there is no sense in shipping them apart, and the refusal of an index at a version a build does not carry costs nothing between binaries that are always the same build.

It opens on **the vaults and what is due in each**. That is the whole of its front door: what a person owes today, and the place to come back to between decks.

### It reads the registry and never writes it

Which vaults exist, where they are and which was opened last is what the registry answers, and it answers before any database is opened.

Every write to it rewrites it whole and nothing locks it between processes, so flashcards recording that it had been started could erase what the editor had just written. It is read here and never written.

### It opens the index for writing

Both pools, the schema put in place, and the same connection settings the editor opens with. The window mints marks, its agent writes cards, and cards are edited during a session; every one of those writes would otherwise leave the index behind, and the window would draw from an index it could not bring up to date. Which preset schedules a deck is among what the index answers, so the staleness would reach how a day is planned and not only what a heading says.

Every write brings the paths it touched up to date before it returns, through the same refresh the editor uses and over the same cutting sizes, so neither application re-cuts what the other wrote. A vault is opened and walked through one path, and both windows go through it.

Two processes now hold a writer each, which is why the write pool begins every transaction immediately, under lock.

A card's heading reaches the index without the mark the card is known by, so the cards themselves are read from the deck files, whatever else is open.

### It writes marks into decks, and works from what it reads back

A card typed by hand carries no mark until the application next writes its file. A person who opens only this application never writes that file in the editor, so their deck would hold cards nothing can be recorded against.

A deck holding a card with no mark is therefore written once, which mints a mark for every card in it, and **the deck is read again afterwards and the session works from that read**.

**It is done when a person sits down to a vault, and never for the counting.** The front door counts every vault the installation holds, and a count that wrote would write into all of them at every launch — including the ones nobody opened. A card with no mark is counted as nothing until the vault it is in is sat down to.

The vault's write lock lives in the process, and two processes on one vault hold a lock each. A mark minted here while the editor is saving the same deck is a write one of the two loses, and the editor's own save mints marks of its own for whatever is missing them. Reading the deck back is what makes that harmless: a card is reviewed under the mark the file holds, so a stamp that did not land is a card left out of this session and stamped again at the next, and no answer is ever recorded against a mark that is not in the file.

### It reviews, and it corrects the card in front of the person

A person who finds mid-session that a card is wrong is holding the one piece of knowledge that fixes it, at the one moment they will never have again. So a card can be added, edited, have a value taken off it, and be removed; and a section can be made, renamed and removed — and nothing else. A deck and a stencil are what a vault is arranged into, and nothing here makes one; no note, link or document is written from this window either.

### The tools are halved by what they do

Every tool family registers its reading half and its writing half separately, and a surface is a set of those halves: a reading surface with no writer on it at all, and the reviewing surface, which is that one and the seven card writers. So a tool gains a behaviour once and every surface has it.

**A surface is a claim about what is absent**, and a test that lists what is present passes with anything extra on it. Every surface is asserted as an exact set. Which tools stand on each is [`../agents.md`](../agents.md).

### The tools it serves announce no port

The address and token an agent a person configured reads name one vault and one window. Two windows writing that file would point that agent at whichever started last, so this window listens on an ephemeral loopback port with a token that lives in memory, writes neither file, and takes no address from the command line.

### The agent follows the vault the person sat down to

The front door counts every vault; a session is on one. The agent is told which vault it works when it is started, and it is stopped when a session opens on another vault or the window closes. A conversation belongs to one card, and answering the card ends it: a thread carried across cards would answer the card in front of the person out of the one behind it.

### Nothing embeds behind this window

A query has no vector and the meaning half of a search does not run. A question whose answer is in a book the person paraphrased may be missed here and found in the editor.

### What the deck is joined to is read from the index

A deck carries every link its cards write, because the markdown parser puts a note's body links on it whatever type the note is. What the panel shows is one query over that, in both directions. Nothing pulls `[[wikilinks]]` out of a card's values and no deck is read again to find them.

The links belong to the deck and not to one card, so the same notes stand behind every card of a deck. Narrowing the list to the card in front of the person would re-resolve what the index already resolved, to shorten a list a person opened on purpose. What the panel leaves out and what it says about a name that resolves to nothing is [`../flashcards.md`](../flashcards.md).

## Consequences

- **A third binary is built, packaged and released.** Both link against the system's own webview, so both are built on the machine they are built for, and the installer carries two windows.
- **A person can run their cards without knowing an editor exists.** That is the point of the front door, and it is why marks are minted here.
- **Two processes write to one vault.** The write lock does not reach across them. The loss it admits is a mark, never an answer, and a lost mark is minted again at the next session.
- **Two windows are open on one vault when a person moves between them**, and each holds its own lock and its own read of the same files.
- **What this window writes, it can see.** A mark it minted and a card its agent corrected answer the next question it asks.
- **A session can change the vault.** It changes cards, which is what the session is about, and every other kind of file is as safe here as it was.
- **Every tool surface has a list that must be edited when a tool is added**, and a build that forgets fails.
- **This window finds less than the editor would on the same question**, and says nothing about it.
- **Two agents can be running on one machine**, one to a window. Both read the whole vault; only the editor's writes notes.
- **The panel's list is the same on every card of a deck**, and long decks have long lists.

## Alternatives considered

**A tab in the editor.** Rejected: every tab there is a thing in the vault — a note, a deck, a stencil, a plex, a conversation — and a review is not a thing in the vault. It is an activity over many of them, and the metaphor does not hold.

**A second window of the editor.** Rejected: reaching the daily act through the application built for the occasional one is backwards. Writing cards is occasional; running them is daily.

**Reaching the vault without the index at all**, by walking it for decks at every launch. Rejected: it is the scanner written twice, and the second copy has no watcher, so it is the scanner written twice and worse.

**Refusing to review a card with no mark**, and leaving marks to the editor. Rejected: it makes the editor a prerequisite for the daily act, which is the arrangement this decision exists to avoid.

**Leaving the index to the editor's next scan.** Rejected: what this window writes is what it draws next, and a person would correct a card and watch the old one come round until they opened another application.

**Asking the editor to rescan.** Rejected: it makes the daily act depend on a process that may not be running, and it is a second way to do what a write already knows how to do.

**Keeping the reader-only handle for everything but the writes.** Rejected: two handles over one file, told apart by which questions they answer, is a rule to remember at every call site for no gain once the file is open for writing anyway.

**Giving the reviewer the surface the editor has.** Rejected: an application whose whole decision is that it reviews would be able to rearrange the vault it is reviewing from.

**Letting it read alone.** Rejected: the moment a person knows a card is wrong is the moment they are looking at it, and sending them elsewhere to fix it means it stays wrong.

**Leaving `note_search` and `source_read` off, so the read-only index need not grow.** Rejected: the objection this exists to answer is a confident wrong answer about a book the person owns, and those two tools are how the book is reached.

**Serving the tools on the editor's port and letting the reviewer borrow them.** Rejected: the reviewer would then need the editor running, which is the arrangement this decision exists to avoid.

**Keeping the conversation across the cards of a session.** Rejected: the context grows through a session and the card behind starts answering for the card in front.

**Narrowing the panel's list to the card in front of the person.** Rejected: it re-resolves what the index resolved, to shorten a list a person opened on purpose.

**Reading every joined note's text.** Rejected: a hub note makes that unbounded, and the bound has to be somewhere a person can see.
