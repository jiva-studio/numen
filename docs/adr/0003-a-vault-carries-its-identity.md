# A vault carries its identity, and application state lives with the application

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core` — `usecase/vault`, `internal/adapter/appstate`, `adapter/settings`
- **Related:** [Files on disk are the source of truth](0001-files-are-the-source-of-truth.md), [One database for all vaults, outside them](0002-one-database-for-all-vaults.md), [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md)

## Context

A vault folder is renamed, moved to another disk, and copied, and the index has to go on pointing at the same vault through all three. The installation also knows things about itself — which vaults exist, which was open last, what the person has chosen — and those belong to no vault at all.

## Decision

### A vault carries its identity inside itself

```
<vault>/.numen/config.json
```

```json
{ "v": 1, "id": "01J8F3K2M9QRSTVWXYZ012" }
```

The identifier is a ULID, generated once, at the moment the person **adds the vault to the application**. That act is what permits the first write into the folder, and nothing generates an identity by scanning. This identity is what a vault reference means throughout the index, and it is how a folder is recognised after it is moved or renamed.

### Two folders carrying one identity are a copy

Adding a folder whose identity is already registered at another path is settled by looking at that path. If it still carries this identity, both folders are there, they are copies of one vault, and the add is refused naming both paths. If it does not, the folder moved and the registry is brought up to date. A person makes a copy into a vault of its own by deleting its service folder and adding it again. See [vaults](../vaults.md).

### One service folder

Everything the application writes into a vault that is not a note goes into one folder, in subfolders. Its name is a setting and the default is `.numen`. It is excluded from indexing whole, by name, whichever name it has.

It holds what the application made and cannot make again: the identity above, and one subfolder to each kind of thing it makes, named after that kind. A kind added later takes a subfolder of its own; nothing shares one, so a kind can be swept away by removing its folder.

### Application state lives with the application, as JSON

What the installation knows about itself lives in the platform's configuration directory, never in a vault and never in the index. It is two files, because it has two authors.

```
<config dir>/numen/vaults.json    the application writes it
<config dir>/numen/numen.json     the person writes it
```

**`vaults.json` is the application's.** Which vaults exist, where they are, and which was open last. Every write rewrites it whole.

**`numen.json` is the person's.** Everything they may want to change, in sections named for the part of the application each is about — see [settings](../settings.md). The application writes this file too, a field at a time, each addressed by the path it stands at. A patch replaces the bytes of a named field where they sit and carries every other byte through, so the order the person arranged their sections in and the way they wrote their numbers come back as they were. A file that does not parse is not written. A file that is not there is created holding the patched fields alone.

### An entry point reads the settings once

An entry point reads the file, says what it found, and hands the result down. Nothing below one opens the file, so nothing below one can reach the machine's own settings, a model, or a paid account.

## Consequences

- Losing a vault's `.numen/config.json` orphans every row filed under it: adding the folder again produces a new identity and a new scan.
- Copying a vault folder copies its identity, and the second folder cannot be added while the first is still there.
- A folder of the application's appears inside the person's vault.
- Losing the registry costs the person re-adding their vaults; losing the settings starts the application on its defaults.
- A person's settings file comes back from a patch arranged the way they left it, and two authors write one file.

## Alternatives considered

**A vault identified by its path.** Rejected: a rename and a move are indistinguishable from a delete at the filesystem level, and a drag in the file manager orphans every row filed under the old path.

**One file holding the registry and the settings together.** Rejected: the application rewrites the registry whole on every change, and a file rewritten whole is no place for something typed by hand.

**Application state in SQLite.** Rejected: a database for a handful of entries, whose most important moment is the one where the application will not start.
