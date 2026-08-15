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

## What the application may add

The complete permitted set, from ADR-0012. Everything must survive a third-party
markdown editor and stay readable to a human.

| Addition | Where | Status |
| --- | --- | --- |
| YAML frontmatter | top of file | allowed; key set not yet fixed |
| `[[wikilink]]` | body | allowed; resolution rules not yet fixed ([#12](https://github.com/jiva-studio/numen/issues/12)) |
| `key:: value` | body | allowed; no consumer defined yet |
| `^anchor` | end of a line | allowed; syntax and scope not yet fixed ([#10](https://github.com/jiva-studio/numen/issues/10)) |

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
| Which frontmatter keys the application owns, and how collisions with the user's own keys are handled | [#4](https://github.com/jiva-studio/numen/issues/4), [#10](https://github.com/jiva-studio/numen/issues/10) |
| How a note is identified, and whether that identity lives in the file | [#10](https://github.com/jiva-studio/numen/issues/10) |
| The shape of the links block, and the roles and types a link carries | [#4](https://github.com/jiva-studio/numen/issues/4) |
| How a `[[wikilink]]` resolves to a target, and what happens when it is ambiguous | [#12](https://github.com/jiva-studio/numen/issues/12) |
| Anchor syntax, alphabet, and whether the user may name anchors | [#10](https://github.com/jiva-studio/numen/issues/10) |
| Card syntax in the body, and how a card keeps its identity across edits | [#9](https://github.com/jiva-studio/numen/issues/9) |
| How attachments are referenced | [#11](https://github.com/jiva-studio/numen/issues/11) |
| Which files in the vault are notes and which are service data | [#5](https://github.com/jiva-studio/numen/issues/5) |
