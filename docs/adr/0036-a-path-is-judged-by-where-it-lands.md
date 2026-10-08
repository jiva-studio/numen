# A path is judged by where it lands

- **Status:** Accepted
- **Date:** 2026-09-05
- **Applies to:** `modules/libs/core`
- **Related:** [A vault carries its identity, and application state lives with the application](0003-a-vault-carries-its-identity.md), [The application writes to the vault](0017-the-application-writes-to-the-vault.md), [An agent reaches the vault through tools](0021-an-agent-reaches-the-vault-through-tools.md)

## Context

Everything that reaches a vault from outside the application — the window, a tool an agent calls, the command line — names a file by its path from the vault root. A path is a spelling, and on every filesystem this runs on a spelling can lead somewhere other than where it reads: a link, a folder swapped for a link between one call and the next, a name differing only in case, a device name on Windows. The vault is the person's, one folder inside it is the application's, and the rules that keep a note out of the second are written against those same paths.

## Decision

### A path is read twice: as it is spelled, and where it lands

A path is cleaned first. One that is absolute, that climbs out, that carries a NUL, or that names the root itself is refused before the filesystem is asked anything. What survives is joined to the root and resolved as deep as it exists, so a file about to be created is judged by the folder it would land in, and a path landing outside the root is outside however it was spelled.

### The vault's and the application's do not overlap, under either reading

A path is the vault's when neither its spelling nor its landing is in the service folder, and the application's when both are. One whose two readings disagree — a link inside the vault leading into the service folder, or the folder itself linked to somewhere outside — is neither's, and both rules refuse it. That the two never both answer is a property held by a test, not something that happens to be true of the call sites there are today: were they to overlap, a writer of notes could be handed a path into the application's own folder, and the refusal that keeps an artifact from being read back as a note would hold only where somebody remembered it.

### A link a person made in their vault is the vault's

Resolving is not a rule against links. A folder of notes reached through one is the vault's, which is the arrangement somebody makes on purpose when a part of their vault is synced or shared. Reading and writing then go to the place rather than to the name, because the place is what a rename replaces and what another tool is watching.

### The write itself goes through a handle on the root

A rule read before a write is a rule about the filesystem as it was a moment ago. Every step of a write is made through a handle opened on the vault root, so a folder swapped for a link while the write is on its way is refused by the machine rather than by a reading that has gone stale.

## Consequences

- Every path from outside enters through one pair of functions, and a call site that reaches the filesystem without them is how this is broken.
- Resolving costs a call to the filesystem for every path, on the read side as well as the write side.
- A vault root that is itself a link is named by what it resolves to, so one folder reached by two names is one vault.
- A path that is refused says which rule refused it and not what is on the machine behind it.

## Alternatives considered

**Judge the spelling and nothing else.** Rejected: cleaning catches `notes/../../etc`, and no amount of reading the text finds a link. The paths that matter arrive from a model or a person, and both write plausible ones.

**Refuse every link inside a vault.** Rejected: the vault is ordinary files and the person arranges it, so a synced folder linked into it is theirs to make.
