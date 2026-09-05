# Note format

The on-disk format of a single note. This is a specification, not a decision record: every rule here traces to an accepted ADR.

Scope is the **note file only**. The layout of the vault around it — the service folder and what is kept there — is in [A vault carries its identity, and application state lives with the application](adr/0003-a-vault-carries-its-identity.md).

## What a note is

A UTF-8 file named `.md`, in any folder of the vault the user likes. A stencil, a deck and a preset are notes and carry the same extension.

Two places are never notes: the service folder — `.numen` by default — and any directory whose name begins with a dot, which holds tool state. Both are skipped whole. The application neither imposes nor rearranges layout.

A markdown file written by anything else — vim, a script, another editor — is a valid note from the first byte. Nothing has to be registered, imported or converted.

## Structure

```
---
<YAML frontmatter, optional>
---

<markdown body>
```

Frontmatter is optional. A note with no frontmatter is a normal note.

## Frontmatter

Fields the application owns live here rather than in the body. The body is prose the user wrote; the frontmatter is where machine-readable facts about the note as a whole belong.

The frontmatter is shared, not owned:

- Keys the application does not own are **preserved verbatim**, in their original order, including keys it may own in a future version.
- Owned keys are a **closed set**, and each is introduced by an ADR that defines its meaning. The set is listed below.
- A **collision is the user's win**: if a user key has the name of an owned key, the application reports the conflict rather than overwriting or reinterpreting it.

### Owned keys

| Key | Meaning | Decided in |
| --- | --- | --- |
| `title` | The name a note is shown by. Written when a note that already has a non-empty one is renamed, and when a title no filename can carry whole is given to one that has none. | [A vault is scanned in the background](adr/0008-a-vault-is-scanned-in-the-background.md) |
| `id` | The identity of the note, a ULID. Written when the application creates a note or changes what is in it, never backfilled and never written by a person typing in it. A note that was moved carries the identifier it carried before. | [A vault is scanned in the background](adr/0008-a-vault-is-scanned-in-the-background.md), [A schema change is a numbered migration](adr/0007-a-schema-change-is-a-numbered-migration.md) |
| `links` | Links that carry a role, and optionally a type, a label and a note. | [Links](links.md) |
| `type` | Which of four this note is: `note`, `deck`, `stencil` or `preset`. A note carrying none is a `note`. | [The stencil, the deck and the card](adr/0027-the-stencil-and-the-deck.md), [A preset is a note, and one arithmetic schedules it](adr/0034-the-preset.md) |
| `fields` | The fields a card cut by this stencil has, in the order they are asked for. Read on a stencil and nowhere else. | [The stencil, the deck and the card](adr/0027-the-stencil-and-the-deck.md) |
| `goal`, `by_date`, `minutes_a_day`, `new_a_day`, `reviews_a_day`, `retention` | What a day of the decks pointing at this preset holds, and which of those closes it. Read on a preset and nowhere else. | [A preset is a note, and one arithmetic schedules it](adr/0034-the-preset.md) |
| `learned`, `interval` | What this preset counts as a card learned, and what a day named under `by_date` is tested by. Read on a preset and nowhere else. | [A preset is a note, and one arithmetic schedules it](adr/0034-the-preset.md) |
| `counts`, `backlog` | What a day's budget is spent on, and in what order. Read on a preset and nowhere else. | [A preset is a note, and one arithmetic schedules it](adr/0034-the-preset.md) |
| `load` | The share of a day's load each day of the week carries. Read on a preset and nowhere else. | [A preset is a note, and one arithmetic schedules it](adr/0034-the-preset.md) |
| `even_load` | Whether a card is moved off the day it fell on. Read on a preset and nowhere else. | [A preset is a note, and one arithmetic schedules it](adr/0034-the-preset.md) |

A note with no `title` is named by its filename. A heading in the prose names nothing: what a person writes in the body is the body, and typing one does not rename the note.

How far a rename reaches — whether a new title renames the file, and whether a renamed file writes the new name into the note — is `naming.sync_title_and_filename` in [Settings](settings.md).

A note carrying no `id` is indexed in full and simply cannot be a *target*: nothing points at it with `note://`, and nothing is attached to it.

## What the application may add

The complete permitted set, from [The note file](adr/0018-the-note-file.md). Everything must survive a third-party markdown editor and stay readable to a human.

| Addition | Where | Status |
| --- | --- | --- |
| YAML frontmatter | top of file | allowed; the keys the application owns are the closed set above |
| `[[wikilink]]` | body | a link with the role `ref`, resolved by name ([Links](links.md)) |
| `{{Field}}` | the body of a stencil | where a card's value goes on a face ([Cards](cards.md)) |
| `^` and a card's mark | the end of a card's heading in a deck | what that card is, wherever it goes ([Cards](cards.md)) |

The first two are every note's. The last two belong to a note of `type: stencil` or `type: deck`, and no other kind of note may carry them.

Nothing else is permitted: no custom fences, no HTML comments carrying data, no sidecar files, no private extension. A kind of note that wants an addition of its own asks for it in a record, as a stencil and a deck did, and the table above is what is kept current.

## Handling of existing files

These follow from files being the source of truth, where the application is a guest in a file the user also edits.

- **The application does not rewrite what it did not change.** Formatting, whitespace, key order in frontmatter and line endings are preserved as found. A save must not produce a diff the user did not ask for.
- **Line endings are preserved per file.** Every break in the file decides. A file whose breaks are all CRLF has its body written with CRLF; any other file has its body written with LF, and a new file is written with LF. The frontmatter arrives on the other side as the bytes it went in as, so a file whose breaks are mixed above the body keeps that mixture and a save that changes no text leaves the file byte for byte as it was.
- **Unknown frontmatter keys are preserved verbatim.** The application reads the keys it owns and leaves everything else untouched, including keys it will own in a future version.
- **A file that fails to parse is not rewritten.** Broken YAML is reported, never repaired in place — repairing it means guessing at content the user wrote.
