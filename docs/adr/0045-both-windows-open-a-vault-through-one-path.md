# ADR-0045: Both windows open a vault through one path

- **Status:** Accepted
- **Date:** 2026-09-02
- **Applies to:** the two desktop applications and the index they share
- **Supersedes:** ADR-0035 — "Nothing here rescans a vault", and its consequence "What a person is told is unchanged: a vault nothing has scanned still reads as one nothing has read"
- **Related:** ADR-0008, ADR-0009, ADR-0020, ADR-0030, ADR-0034

## Context

The index is level with a vault only for as long as nothing has been edited (ADR-0008), and the editor's answer is a walk of the vault on every opening. The review window had no walk. It drew from whatever the index last held and, for a vault the index did not carry, told a person to open the editor once.

A vault is edited on another machine and arrives by sync. Nothing on this machine has run since. The watcher covers a window's own life (ADR-0009), so everything that arrived while nothing was running went past it: new decks are missing, removed cards are still asked, and an edited card is asked in its old words. Review is the daily act and the editor the occasional one (ADR-0030), so this is the ordinary case and not the corner.

The review window is no longer the lesser of the two. It writes the index (ADR-0035), it mints marks, its agent writes cards, and it schedules a day. Both applications now hold a vault open, and each had grown its own way of doing it: the editor watched a vault through the use case that reindexes what changed, and the review window watched one through the raw watcher and threw the paths away.

## Decision

### Opening a vault is one thing, in the container

The container assembles the walk that brings the index level with a vault, the watch that keeps it level while it is open, and the levelling of the paths a write touches. Both windows open a vault through it, so a vault opened in one is a vault opened in the other.

The watch is started before the walk, and the notes written underneath the walk are read again after it: a walk writes in groups from what it read, so its copy of a note lands last however early the note was read.

**What each window says about an opening is its own.** How far the walk has got, whether the vault is ready to draw, and what a person is told when it fails belong to the window that has the vault open.

**What is not the same is not shared.** The editor keeps its documents, its recognitions, its transcriptions, its embedding passes and its tabs. The review window keeps its counts, its sittings and the watch it holds on the index file, which it needs because the other window writes that file.

### The vault is read when the window opens it

Counting a vault in the review window walks it into the index first, and the count follows the walk. A walk skips a file whose size and modification time did not change, so an unchanged vault costs a walk of the tree — what the editor already pays on every launch.

Each vault is walked once for the life of the window, and the watch carries it from there. One vault is walked once at a time however many counts ask for it, and a walk that failed says why in that vault's row and is not begun again until something moves underneath the window.

### Nothing is counted from a walk half done

While a vault is being walked, its row holds no count and says it is being read. The numbers arrive with the count the finished walk wakes.

A walk writes in groups and takes what vanished out at the end, so an index part-way through a walk answers with some of the vault's decks and not others. A number counted from that is wrong, and a row that says it is working is not.

### What is being done is said while it is done

Reading a vault is work, in the list of work each window keeps, named by the vault and by how many notes have been written.

A vault the index does not carry has nothing to draw until the walk is done and the person is waiting on it, so that work is drawn the moment it begins. A vault the index carries is work drawn once it has lasted, and a walk that finds nothing changed passes without a word.

### A vault is unread only while nothing has read it

`ErrUnread` is the index's answer that it does not carry a vault. It reaches a person only where nothing reads one: a build that names no walk, and a sitting opened on a vault whose walk has not finished.

## Consequences

- **What a person is asked is what the vault says now.** A vault edited anywhere, by anything, is read before its cards are counted.
- **A change made while the review window is open reaches the index.** Its watch reindexes what changed, so a deck written beside it is counted from an index that holds it.
- **Opening the review window costs a walk of every vault it lists.** On an unchanged vault that is a walk of the tree and no file read; the rows say they are being read while it runs.
- **Two processes may walk one vault at once.** Their writes queue in SQLite as every other pair of writes does, and a fingerprint that did not change is a file neither of them reads twice.
- **A count asked for while one is running is answered by one of its own.** A walk finishes after the count has worked its row out, so the count that walk wakes is not folded into the one already running.
- **A window opening a vault a third way is a change in one file.** Both windows are held to the same opening by calling it, and not by being kept in step.

## Alternatives considered

**Walking only a vault the index does not carry.** Rejected: the index carrying a vault says nothing about the vault being level with it, and what arrived while nothing was running is exactly what no watcher saw.

**Drawing the counts as they stood while the walk runs.** Rejected: a walk part-way through has some of the vault's decks in the index and not others, and a number counted from that is one a person would act on.

**Walking at the window's start rather than at the count.** Rejected: the count is where a vault's numbers are made, and a vault added to the list while the window is open is one a walk at the start would not read.

**Lifting the whole of the editor's opening, its passes over sources and vectors among it.** Rejected: reading books and making vectors are the editor's, and a shared thing holding what one caller never uses is two paths written as one.
