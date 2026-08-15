# ADR-0011: Links are written by name; `note://` is the auxiliary form

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** ADR-0003, ADR-0009, ADR-0012

## Context

ADR-0009 gives every application-created note a stable identifier, which makes an
identifier-based link possible: `note://01J8F3K2M9QRSTVWXYZ012`. The tempting
conclusion is that identifiers are the correct way to link and names are a
convenience.

That conclusion is wrong, and the argument is worth writing down because it is
not obvious.

Inside the application the two forms behave identically: renaming a note makes
the application walk the vault and rewrite the name everywhere it resolved to
that note. That touches many files — which is exactly what Obsidian does, and
nobody complains.

The difference appears **only on a rename made outside**, with the application
closed. Then `[[Thermodynamics]]` goes dangling and `note://…` survives. And that
has to be priced honestly against the position this project already takes:
renaming things behind the application's back is the user's own doing. Under
that position the advantage nearly vanishes, while the cost — unreadable text in
files people read in vim — is paid on every line, forever.

## Decision

**Names are the primary form.** `[[Thermodynamics]]` is normal and expected: it
is readable, typeable, and requires knowing no identifiers.

**`note://<id>` is auxiliary**, not "correct". It appears only where it arises by
itself:

1. the application inserted the link — autocomplete, drag and drop, a link made
   in the interface — and already knew the identifier;
2. the name is ambiguous and the priority rules below do not pick one target.

The second is the only case where an identifier is genuinely *needed* rather
than merely convenient.

### `label` for readability

An identifier link may carry a label for whoever reads the file:

```yaml
links:
  - to: "note://01J9K4M5N6PQRSTVWXY7"
    label: "Thermodynamics"
    role: parent
```

`label` takes **no part in resolution** and is not the source of truth for
anything. A stale label is not an error — the address is an identifier, so the
link still goes where it should, and the application refreshes the label the next
time it writes that file. This has to be said out loud, or someone eventually
writes code that falls back to resolving by label when `note://` misses.

### `name://` never appears in a file

It is the internal representation, the value stored in the index. The parser sees
`[[Thermodynamics]]` and stores it as `name://Thermodynamics` so that **every
stored address has a scheme**.

The name is stored exactly as written — a rename has to preserve the form the
user chose — and **compared without regard to case**, because people type
`[[entropy]]` and the file is called `Entropy.md`. Folding at comparison rather
than at parse keeps both.

The prefix is not there to mark something as a name; it is there so parsing is
always a split on `://`, with no rule of the form "no known prefix means a name".
That rule breaks on a title like `Lecture 3: entropy`, which looks like an
unknown scheme.

### How a name resolves

In descending priority:

1. an exact match of the path from the vault root;
2. relative to the folder of the note the link is in;
3. a single filename match anywhere in the vault;
4. several matches — the link is **ambiguous**, not dangling. It resolves to the
   nearest match in the tree, and the ambiguity is reported in the "vault
   problems" view.

Resolution is a **query, not stored state**: adding a file can resolve a link
that was dangling, and deleting one can break a link that worked. The index
stores what was written and what it currently resolves to, and the second is
recomputed rather than trusted.

### A link by name stays inside its vault; a link by identifier does not

A **name** means something only within one vault. Two vaults may each hold an
`Entropy.md`, and neither is the other's answer, so a name resolves against the
vault the link was written in and nowhere else.

An **identifier** is globally unique, so it needs no new syntax to point across
a boundary: `note://…` already names one note in the world. A link written that
way resolves in whichever connected vault holds that note, and the vault it
landed in travels with the answer.

This is the one place where a query deliberately looks past the vault it was
asked about. Everywhere else a missing `vault_id` is the silent bug ADR-0002
warns about; here it is the point, and it is spelled out in one query rather
than left to a filter someone forgot.

**A note in a vault that is not connected has a third state.** Not resolved, and
not dangling either — the application cannot tell a deleted note from one in a
vault it cannot see, and pretending otherwise would report a broken link that is
perfectly fine on the machine where both vaults are open. The state carries its
own reason: *no connected vault holds this note*.

Nothing else follows from it. There is no server, no federation and no fetching:
a link into a vault the user has not added is text with a known shape, and the
interface says so rather than guessing.

The link is only created deliberately — by pointing at a note the user has open
in another vault — so the seam between vaults exists exactly where they put it.

### Renaming a note

Done by the application when the rename happens inside it. An external rename is
not tracked and leaves names dangling — accepted, as in Obsidian.

1. **`note://` links are untouched.** An identifier is not tied to a name.
2. **Only the names that resolved to this note are rewritten** — not every
   textual match. With both `Projects/Entropy.md` and `Archive/Entropy.md`
   present, a link from a neighbouring folder may have meant the other one.
3. **The written form is preserved.** `[[Entropy|entropy's]]` and
   `[[Projects/Entropy]]` change only their target part; the alias and the style
   survive.
4. **Resolution is recomputed across the vault afterwards.** A rename changes the
   picture for other links too: if there was one `Entropy.md` and now there are
   none, links that pointed at it may quietly start resolving to a same-named
   file elsewhere.

## Consequences

**Positive**

- Files stay readable and hand-editable, which is the whole point of ADR-0001.
- Someone who never opens the application still writes links that work.
- The identifier form stays available exactly where it earns its keep.

**Negative**

- Resolution is a query, so the set of dangling links changes as files appear and
  disappear, and the interface has to represent that rather than cache it.
- A rename is an expensive vault-wide operation followed by a re-resolution pass.
- Ambiguity is a permanent condition to be shown, not an error to be fixed once.

## Alternatives considered

**Identifiers as the primary form.** Rejected: it buys protection only against
external renames — which this project has already decided are the user's own
affair — and pays for it with unreadable text in every file, on every line.

**Names only, with no `note://` at all.** Rejected: an ambiguous name has no
other resolution, and a link the application inserts would have to throw away an
identifier it already holds.

**No scheme prefix for names in the index.** Rejected: `Lecture 3: entropy`
parses as an unknown scheme, and "no prefix means a name" is not safely
decidable.
