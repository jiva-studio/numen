# ADR-0003: A link is one object carrying a role and an optional type

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** ADR-0009, ADR-0010, ADR-0011

## Context

Two of the three tools this product merges have a link model, and they are not
the same model. Obsidian has one untyped link, `[[wikilink]]`, and derives
everything from it. TheBrain has a typed navigational structure — parents,
children, jumps — which is what makes its graph navigable rather than a hairball.

Both are needed: the untyped link people already write, and structure deliberate
enough to navigate. The failure to avoid is two parallel link systems, one in the
body and one in frontmatter, that have to be reconciled — or a model where the
semantic label and the navigational meaning live apart and drift.

## Decision

**A link is one record in one place, and its type cannot be stated without its
role.**

| Field | Required | Values | Purpose |
| --- | --- | --- | --- |
| `to` | yes | a link target (ADR-0011) | what it points at |
| `role` | yes | `parent` \| `child` \| `jump` \| `ref` \| `attachment` | navigation and rendering |
| `type` | no | open vocabulary (`requires`, …) | semantics a particular feature reads |
| `label` | no | a few words | what the relationship is called, written on the line that draws it |
| `note` | no | a short string, or a link to a note about the link | why it exists |

`label` and `type` answer different questions and are not alternatives. A type
is a category code reads — one feature, one type. A label is a few words the
person wrote for themselves, drawn along the line between the two notes and read
by nobody but them. A link may carry either, both or neither.

`role` is a **closed** list: navigation and rendering read it, so an unknown role
has no behaviour. `type` is open, and governed by one rule:

> **A type is introduced together with the code that reads it. One feature, one
> type.**

At the start that means `requires`, which the prerequisite-aware scheduler will
read, and nothing else. A vocabulary nobody reads is decoration that becomes an
obligation the moment a vault is full of it.

## Rules of the model

- **`parent` and `child` are one edge.** `parent: B` written in A and `child: A`
  written in B describe the same thing, and the user writes whichever end is
  convenient.

  Both ends are visible today because a link is answered in both directions: what
  a note points at, and what points at it. **Folding them into one stored
  direction is deliberately deferred** until a traversal needs it — a graph walk
  is what turns "the same thing written twice" into a problem, and there is no
  traversal yet. When one arrives, this is where the canonical direction gets
  decided.
- **Multiple parents are ordinary.** The hierarchy is a DAG, not a tree.
- **Cycles in `parent` edges are forbidden**, and arrive anyway. Created inside
  the application, a cycle is refused; arriving from an external edit, it must
  not stop indexing — it goes to the "vault problems" view, and every traversal
  is written to survive one regardless of what was validated.
- **A contradiction is a data problem, not a write error.** If A calls B its
  parent and B calls A its parent, the application neither refuses nor picks a
  side: it shows the conflict. This follows from ADR-0001 — files are edited
  outside, so invalid states arrive fully formed and must be *displayable*
  rather than *preventable*.
- **`sibling` is not stored.** Siblings are children of a shared parent: a
  query, not data.
- **`role='ref'` is the ordinary `[[wikilink]]`** in the body, extracted by the
  parser and never hand-authored as markup.
- **The same link written twice is one link**, and the annotated record wins: a
  wikilink in the body that is also described in `links:` keeps the role, type
  and note from `links:`. Sameness is decided by where the links resolve, not by
  how they were typed — `[[notes/Entropy]]` and `[[Entropy]]` are one link when
  they land on one note.
- **`role='attachment'`** points at something that is not a note (ADR-0010). It
  takes no part in the hierarchy and is excluded from cycle checking.

## Example

```yaml
---
links:
  - to: "[[Thermodynamics]]"
    role: parent
  - to: "[[Linear algebra]]"
    role: parent
    type: requires
    note: "only eigenvectors are needed"
  - to: "[[Entropy as disorder]]"
    role: jump
    type: contradicts
---

Ordinary text with an [[ordinary link]] — that is role=ref, found by the parser.
```

## Consequences

**Positive**

- Navigation, rendering and algorithms read one table.
- The graph is navigable because of roles and not rigid because types are open.
- Writing a relationship from either end means the user never has to open the
  other file to record it.

**Negative**

- Normalisation and deduplication must be deterministic: two machines indexing
  one vault have to produce the same edges, or a synced index disagrees with
  itself.
- The "vault problems" view is not optional polish. Without it a contradiction
  is invisible, and the model looks like it silently lost data.
- One closed list of roles means adding a sixth is a decision about navigation,
  not a convenience — which is the point, and will feel like an obstruction the
  first time someone wants one.

## Alternatives considered

**Type only, no role.** Rejected: nothing tells the interface how to draw or
traverse a link, so every consumer invents its own mapping from type to
behaviour, and they disagree.

**Role only, no type.** Rejected: the prerequisite scheduler and links that
carry an argument need semantics a role cannot hold.

**Separate storage for hierarchy and for semantic links.** Rejected: the same
pair of notes then appears in two systems, which drift.

**Storing `sibling`.** Rejected: derivable, and storing it means maintaining it
on every change of parent.
