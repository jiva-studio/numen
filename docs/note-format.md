# Note format

The on-disk format of a single note. This is a specification, not a decision
record: every rule here traces to an accepted ADR, and where a decision has not
been taken yet the gap is marked rather than filled in.

Scope is the **note file only**. The layout of the vault around it — the service
folder, assets, artifacts written by submodules — is a separate document, written
when the decisions it depends on exist.

## What a note is

A UTF-8 markdown file with the `.md` extension, in any folder of the vault the
user likes. The application neither imposes nor rearranges layout (ADR-0001).

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
  its meaning. The set is listed below and is currently empty.
- A **collision is the user's win**: if a user key has the name of an owned key,
  the application reports the conflict rather than overwriting or reinterpreting
  it.

### Owned keys

| Key | Meaning | Decided in |
| --- | --- | --- |
| — | none yet | |

The note's identifier will live here — that placement is settled (ADR-0012).
Its format, when it is written, and what happens on rename are not
(ADR-0009). The links block is
likewise expected here and not yet defined
(ADR-0003).

## What the application may add

The complete permitted set, from ADR-0012. Everything must survive a third-party
markdown editor and stay readable to a human.

| Addition | Where | Status |
| --- | --- | --- |
| YAML frontmatter | top of file | allowed; key set not yet fixed |
| `[[wikilink]]` | body | allowed; resolution rules not yet fixed (ADR-0011) |
| `^anchor` | end of a line | allowed; syntax and scope not yet fixed (ADR-0009) |

Nothing else is permitted: no custom fences, no HTML comments carrying data, no
sidecar files, no private extension.

## Handling of existing files

These follow from files being the source of truth (ADR-0001), where the
application is a guest in a file the user also edits.

- **The application does not rewrite what it did not change.** Formatting,
  whitespace, key order in frontmatter and line endings are preserved as found.
  A save must not produce a diff the user did not ask for.
- **Line endings are preserved per file.** A file that arrives with CRLF keeps
  CRLF. New files are written with LF.
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
| Which frontmatter keys the application owns, and how collisions with the user's own keys are handled | ADR-0003, ADR-0009 |
| How a note is identified, and whether that identity lives in the file | ADR-0009 |
| The shape of the links block, and the roles and types a link carries | ADR-0003 |
| How a `[[wikilink]]` resolves to a target, and what happens when it is ambiguous | ADR-0011 |
| Anchor syntax, alphabet, and whether the user may name anchors | ADR-0009 |
| Card syntax in the body, and how a card keeps its identity across edits | ADR-0008 |
| How attachments are referenced | ADR-0010 |
| Which files in the vault are notes and which are service data | ADR-0004 |
