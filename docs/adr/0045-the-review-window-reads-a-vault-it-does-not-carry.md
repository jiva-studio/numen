# ADR-0045: The review window reads a vault the index does not carry

- **Status:** Accepted
- **Date:** 2026-09-02
- **Applies to:** the review window and the index it shares with the editor
- **Supersedes:** ADR-0035 — "Nothing here rescans a vault", and its consequence "What a person is told is unchanged: a vault nothing has scanned still reads as one nothing has read"
- **Related:** ADR-0008, ADR-0009, ADR-0020, ADR-0030, ADR-0034

## Context

The list of decks is the index's answer, and a vault absent from the index has none. The review window drew that absence as a sentence across the whole window telling a person to open the editor once.

Review is the daily act and the editor is the occasional one (ADR-0030). A person who has only ever run their cards is being sent to an application they have no other reason to open, to perform a step nothing about their vault required.

The window already opens the index for writing (ADR-0035), already watches the vaults and the index, and already levels every path it touches. Reading a whole vault is the one thing it could not do, and it is the one thing standing between a vault and its cards.

## Decision

### A vault the index does not carry is read here

The count of a vault the index does not carry starts a walk of that vault into the index, and its numbers arrive with the count that finished walk wakes.

It is the same walk. The container assembles one scan — the readers, the note repository cut at the installation's sizes, the fingerprints, and the index's maintenance — and the editor, the command line and this window each take it from there. A vault read in one application is a vault read in every one.

One vault is walked once at a time however many counts ask for it. A walk that failed says why in that vault's row and is not begun again until something moves underneath the window.

### A vault the index carries is left alone

Only absence is answered. A vault the index carries is kept level by the watcher this window runs for as long as it is open, and by every write it makes; walking one again on every opening would spend a walk of the whole vault on the daily act to find what is already there.

### What is being done is said while it is done

Reading a vault is work, in the one list of work the window keeps, named by the vault it is on and by how many notes it has written. It is work a person is waiting on, so it is drawn the moment it begins rather than once it has lasted.

The vault's own row says it is being read and holds no count. A row with a number nothing has counted is a row that lies.

### A vault is unread only while nothing has read it

`ErrUnread` is the index's answer that it does not carry a vault. It is the signal to read that vault, and it reaches a person only where nothing reads one: a build that names no walk, and a sitting opened on a vault whose reading has not finished.

## Consequences

- **A vault reaches its cards without another application.** Adding a vault and opening this window is the whole of it.
- **A first opening on a large vault takes as long as a walk of it.** What is on screen throughout is the list, the vault's row saying it is being read, and the work said in the corner.
- **Two processes may walk one vault at once.** Their writes queue in SQLite as every other pair of writes does, and a fingerprint that did not change is a file neither of them reads twice.
- **A vault edited while nothing was running is still stale.** The watcher covers the window's own life, and bringing a carried vault level is the editor's walk or an asked-for one.

## Alternatives considered

**Walking every vault the window opens on.** Rejected: the walk of a carried vault finds what the watcher and this window's own writes have already put there, and it is paid on the act performed daily.

**Walking at the window's start rather than at the count.** Rejected: a vault added while the window is open is one it would not read, and the count is where the absence is discovered.

**Leaving the sentence and reading the vault behind it.** Rejected: a person told to open another application while this one is reading the vault is being told something that is not so.
