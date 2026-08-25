# ADR-0026: One name per concept

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** the product as a whole
- **Related:** ADR-0004, ADR-0005, ADR-0023

## Context

The same system is written down four times over: in the domain, in the storage, on the wire between the core and a client, and in what the person reads. Each was written at a different time, and nothing has required them to agree.

## Decision

### The product keeps one ubiquitous language

The practice is Evans's **ubiquitous language**: one language spoken by the code, the documents and the people talking about them, so that nothing has to be translated in order to be understood. It holds across the domain, the storage, the wire and the interface, and in the strings a person reads.

### A concept has one name

Where the domain, the storage, the wire and the interface talk about one thing, they use one word for it. A field renamed on the way across a boundary is a defect unless this document lists the rename.

### A word means one thing

A word already spent is spent. A homonym is worse than a synonym: a synonym announces itself, and a homonym lets the reader carry the wrong meaning across with nothing to tell them.

### The listed boundary renames are two

**`calling` and `doing`.** A step is `calling` in the core and `doing` on the wire. The core says what the agent is doing, and the wire is read by something drawing a line about it.

**`stretch` and `span`.** A *stretch* is a run of a source's text where it stands: a start and a length, in bytes over the text the source is read as. A *span* is that same run as a client counts it: `from` and `to`, in UTF-16 code units. A window counts text its own way, and nothing crosses that boundary unconverted.

Both are renames the boundary requires, and the boundary itself is ADR-0005's. A third name for either is a defect.

### The listed homonym is one

**`note`, twice.** A note is a file. A link's `note` is why the link exists, in the person's words. This one is kept: the key is in the file format, where it is read by people and reads naturally, and moving it breaks every vault for a word nobody outside the file ever says.

### The one scoped exception

The interface may not know the vault (ADR-0023). Where a domain word would teach it something it must not know, it takes a word of its own — and that word is in the vocabulary too, meaning that one thing.

### The vocabulary is one page

Every term, with what it must never be called, is in [`../glossary.md`](../glossary.md). A new term is added there in the change that introduces it.

## Consequences

- A name is looked up, not settled by taste at each new file.
- Nothing checks a name against the glossary. A drifted word is found by a reader.
- Renaming a word already in a file format costs every vault, so some collisions are kept and written down.

## Alternatives considered

**A dialect per boundary — a storage vocabulary, a wire vocabulary, an interface vocabulary.** Rejected: a word that cannot survive a boundary is the wrong word, and three vocabularies are three places for one concept to drift.

**Record the collisions and rename nothing.** Rejected: a list of known ambiguities that nobody acts on is a list that grows, and the reader goes on translating in their head.
