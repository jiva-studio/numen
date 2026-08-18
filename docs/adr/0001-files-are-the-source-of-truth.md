# ADR-0001: Files on disk are the source of truth

- **Status:** Accepted, except where noted below
- **Date:** 2026-08-15
- **Applies to:** the product — every application in this repository
- **Partly superseded by:** ADR-0032 — items 2 and 3 of the honest list below, and
  the conflict copy of item 4: what recognises a write coming back, what
  reconciliation covers, and the copy that is not written
- **Related:** ADR-0000, ADR-0002

## Context

This is a tool people invest years into. The largest trust question such a tool
has to answer is what happens to the data if the application breaks, stops being
maintained, or stops being something the user wants to use.

There are two workable answers. Either the application owns a database and
exports files on demand, or the files themselves are the primary storage and the
database is derived. The first is simpler to build. The second is verifiable by
the user: they can open a terminal, run `rg`, edit a note in vim, commit the
vault to git — with the application closed — and nothing is lost.

## Decision

**The vault is an ordinary directory of ordinary files. The application stores
nothing unique in its own database.**

Deleting the index in full must never cause data loss.

- Notes are `.md` files with YAML frontmatter, in whatever folder structure the
  user likes. The application neither imposes nor rearranges layout.
- Link markup lives in the note file.
- Everything derived lives outside the vault (ADR-0002).
- Everything the application produces that cannot be derived lives inside the
  vault as text (ADR-0000).

## Consequences

**Positive**

- Third-party tools work on live data, not on an export: vim, git, ripgrep,
  other markdown editors, file sync, the user's own scripts.
- "You are not locked in" becomes checkable rather than promised.
- Backups and history are whatever the user already uses for files.

**Negative — the honest list**

Files as the source of truth buys the above at the price of exactly these
problems, all of which have to be implemented:

1. **A file watcher.** The application must react to changes it did not make.
2. **Echo-loop protection.** Writing a file makes the watcher report that write
   back. The write path has to recognise its own bytes and drop the event, or
   every save triggers a reparse and, worse, a re-save.

   > **Partly superseded by [ADR-0032](0032-the-window-saves-a-note-as-it-is-typed.md).**
   > No write path recognises its own bytes and no event is dropped: a save comes
   > back through the watcher and the note is parsed again. The re-save is what
   > does not follow — a tab reads its file back and finds the normalised text it
   > already shows, and equal text is not a change.

3. **External edits mid-session.** A note open in the editor can change under the
   user; the application must reconcile rather than overwrite.

   > **Partly superseded by [ADR-0032](0032-the-window-saves-a-note-as-it-is-typed.md).**
   > Nothing is reconciled. A tab with nothing unsaved reads its file again and shows
   > what it holds; a tab with unsaved text stops saving, says which file moved under
   > it, and waits for the person to keep their prose or take the file's.

4. **Conflict copies.** When the buffer is dirty *and* the file changed on disk,
   reconciliation is impossible and a conflict copy is written. No silent winner.

   > **Partly superseded by [ADR-0032](0032-the-window-saves-a-note-as-it-is-typed.md).**
   > No copy is written. There is no silent winner either: the save stops, the tab
   > says so, and the person picks the prose that survives.

5. **Metadata outliving its files.** Rows can survive the file they describe, so
   startup has to detect that.

This list is the entire cost. Parsing markdown, resolving links and merging
concurrent edits are identical in both designs — worth stating, because the two
options are usually compared as though the database design avoided those too.

## Alternatives considered

**Database as the source of truth, files exported on demand.** Strictly simpler:
none of the five problems above exist. Rejected, because it kills the thing that
motivates the design — third-party tools operating on live data rather than on a
copy.

**Files as the source of truth, but only editable while the application is
closed.** Rejected: reintroduces lock-in through the back door, and is
unenforceable.

## Note

External renames are indistinguishable from delete-then-create at the filesystem
level. That is the direct reason note identity cannot be the path — see ADR-0009.
