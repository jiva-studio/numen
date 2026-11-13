# Intent: Plex shows the links written in a note's text as references

**Task Slug:** `references-1839`
**Date:** `2026-11-13`
**Issue:** #1839 (stacked on #1837, the clusters PR; its PR targets that branch)

## 1. Problem & JTBD

- **Pain & Trigger:** A note's text links to other notes with `[[...]]`, and other notes link to it. The plex shows none of these links unless the same note is also a parent, a child or a jump.
- **Target User & Scenario:** A person who writes prose with links and opens the plex of a note to see what it is connected to.
- **Desired Outcome:** Every note the focus links to in its text, and every note that links to the focus, appears in the plex as a reference, in a group of its own, drawn so that it is told apart from parents, children, jumps and siblings, with an arrow that says who links to whom.

## 2. Scope

### In scope

- A fifth kind of related note, the reference: a note linked from the focus' text, or linking to the focus in its text.
- Placement: references form a cluster above and to the left of the focus; jumps form a cluster below and to the left. Two clusters, always, no option.
- Look: a reference has its own color and its own pill background. Its line carries an arrow: from the focus to the note when the focus links to it, from the note to the focus when the note links to the focus.
- A plugin setting **References** with four values: off, outgoing, outgoing and incoming, incoming only. Default: outgoing and incoming.
- A note that is already a parent, a child or a jump of the focus keeps that relation and is not repeated as a reference.
- Only `[[...]]` links count.

### Out of scope

- `[text](note.md)` links (not indexed today).
- Related notes of related notes (separate issue).
- Creating a reference by dragging (see open questions in the spec).
- Anything about how the groups are packed: that is #1837.

## 3. Limits & Failure Modes

| Scenario | Expected behavior |
|---|---|
| A note has no text links and no backlinks | The plex is drawn as before this change. |
| Setting off | No reference is shown; the plex is drawn as before. |
| Outgoing only / incoming only | Only the references in that direction are shown. |
| A link to a note that does not exist or leaves the vault | Not shown, as other dangling links today. |
| A note links to itself | Not shown. |
| Many references | They are counted in "not shown" like any other group that does not fit. |
| A note that is both parent/child/jump and linked in the text | Shown once, under the explicit relation. |

## 4. Invariants

- With no text links in the vault, or the setting off, the plex looks as it does today.
- A reference is never created, changed or removed by the plex: it is read from the text.
- Notes never overlap; no change to any other relation.
- Pressing, opening and the menu of a reference note work as for any other note.

## 5. Decisions of the owner

- Reference is its own relation kind; contract as in the issue (relation value, direction value on the related note, explicit relations win).
- Setting values and default as above; arrow shows direction.
- Cluster above left of the focus, jumps below left.
- Go and TypeScript plugin SDKs keep the same shape.
- A note linked both ways is one reference with arrowheads at both ends, shown in every mode but off.
- A sibling wins over a reference; a drag to the left stays a jump; frontmatter `ref` entries count; no Go SDK call now.
