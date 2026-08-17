# Note format

The on-disk format of a single note. This is a specification, not a decision
record: every rule here traces to an accepted ADR, and where a decision has not
been taken yet the gap is marked rather than filled in.

Scope is the **note file only**. The layout of the vault around it — the service
folder, assets, artifacts written by submodules — is a separate document, written
when the decisions it depends on exist.

## What a note is

A UTF-8 file whose extension is one of those configured as notes, in any folder
of the vault the user likes. The default is `.md` alone (ADR-0012).

Two places are never notes: the service folder — `.numen` by default (ADR-0013) —
and any directory whose name begins with a dot, which holds some tool state
rather than anything a person wrote. Both are skipped whole. The application
neither imposes nor rearranges layout (ADR-0001).

A markdown file written by anything else — vim, a script, another editor — is a
valid note from the first byte. Nothing has to be registered, imported or
converted (ADR-0012).

## Structure

```
---
<YAML frontmatter, optional>
---

<markdown body>
```

Frontmatter is optional. A note with no frontmatter is a normal note.

## Frontmatter

Fields the application owns live here rather than in the body (ADR-0012). The
body is prose the user wrote; the frontmatter is where machine-readable facts
about the note as a whole belong.

The frontmatter is shared, not owned:

- Keys the application does not own are **preserved verbatim**, in their original
  order, including keys it may own in a future version.
- Owned keys are a **closed set**, and each is introduced by an ADR that defines
  its meaning. The set is listed below.
- A **collision is the user's win**: if a user key has the name of an owned key,
  the application reports the conflict rather than overwriting or reinterpreting
  it.

### Owned keys

| Key | Meaning | Decided in |
| --- | --- | --- |
| `title` | The name a note is shown by. Read, never written: the application does not add one and does not rewrite one it finds. | ADR-0012 |
| `id` | The identity of the note, a ULID. Written when the application creates a note, moves it or changes its links, never backfilled and never written by a person typing in it. | ADR-0009, ADR-0032 |
| `links` | Links that carry a role, and optionally a type, a label and a note. | ADR-0003 |

A note with no `title` is named by its first level-one heading, else by its
filename. The order is fixed so that a name does not move between versions.

A note carrying no `id` is indexed in full and simply cannot be a *target*:
nothing points at it with `note://`, and nothing is attached to it (ADR-0009).

## What the application may add

The complete permitted set, from ADR-0012. Everything must survive a third-party
markdown editor and stay readable to a human.

| Addition | Where | Status |
| --- | --- | --- |
| YAML frontmatter | top of file | allowed; key set not yet fixed |
| `[[wikilink]]` | body | a link with the role `ref`, resolved by name (ADR-0011) |
| `^anchor` | end of a line | allowed; syntax and scope not yet fixed (ADR-0009) |

Nothing else is permitted: no custom fences, no HTML comments carrying data, no
sidecar files, no private extension.

## Handling of existing files

These follow from files being the source of truth (ADR-0001), where the
application is a guest in a file the user also edits.

- **The application does not rewrite what it did not change.** Formatting,
  whitespace, key order in frontmatter and line endings are preserved as found.
  A save must not produce a diff the user did not ask for.
- **Line endings are preserved per file.** A file whose endings are uniformly
  CRLF keeps CRLF. New files are written with LF. A file whose endings are mixed
  is written with LF, so it is made uniform the first time it is saved.
- **Unknown frontmatter keys are preserved verbatim.** The application reads the
  keys it owns and leaves everything else untouched, including keys it will own
  in a future version.
- **A file that fails to parse is not rewritten.** Broken YAML is reported, never
  repaired in place — repairing it means guessing at content the user wrote.

## Not decided yet

These are open, and this document will be extended as each is settled. Nothing
below should be implemented from guesswork.

| Question | Where it is decided |
| --- | --- |
| Card syntax in the body, and how a card keeps its identity across edits | ADR-0008 |
| What a submodule may write into the service folder | ADR-0004 |

Anchors are decided (ADR-0009) and not yet read: nothing attaches to a block, so
the parser leaves `^anchor` as the text it is.
