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

### A word means one thing inside its context

A word means one thing within a bounded context. The same word naming something else in another context is not a defect: a reader is inside one context at a time, and there the meaning is unambiguous. A homonym inside one context is a defect, because nothing there tells the reader which meaning arrived.

The contexts are the vault and what is written in it, the index and search, cards and review, recordings and transcription, the agent, the installation, and the window. The installation is what a person configures and the vaults this machine holds; the window is what is open in front of them, what is being done behind it, and how it goes. The interface draws all of them and is none of them.

### A concept takes the name its field already gives it

Where speech recognition, spaced repetition, information retrieval, Markdown, WebVTT, EPUB, the Model Context Protocol, protobuf, SQLite or the web platform already names a thing, the code uses that name. A recording is transcribed; an icon is an icon.

A word is never coined to keep clear of a word used elsewhere. Where the name a field gives a thing is taken in another context, it is taken again here, and the contexts are what make that safe. A coined word is for a concept this product has and nothing else names, and it is defended in its glossary entry.

### The listed boundary rename is one

**`calling` and `doing`.** A step is `calling` in the core and `doing` on the wire. The core says what the agent is doing, and the wire is read by something drawing a line about it. The boundary is ADR-0005's.

### A stretch and a span are two things

A *stretch* is a run of a source's text where it stands: `Start` and `Length`, in bytes over the text the source is read as. A *span* is that same run as a client counts it: `From` and `To`, in UTF-16 code units. A window counts text its own way, and nothing crosses that boundary unconverted.

Each keeps its own name wherever it is written — in the core, in the storage, on the wire and in the interface — and a type carrying `Start` and `Length` is a stretch whatever it is called. A third name for either is a defect.

### The homonym inside one context is one

**`note`, twice.** A note is a file. A link's `note` is why the link exists, in the person's words. Both are the vault's, so this one is a homonym the rule above forbids. It is kept: the key is in the file format, where it is read by people and reads naturally, and moving it breaks every vault for a word nobody outside the file ever says.

### The interface speaks the interface's language

The interface may not know the vault (ADR-0023). What it draws is a row, an icon, a label, a badge, and it says so: the word interface design and the web platform already use, never one coined for the occasion.

Where the thing it draws is a domain concept, it says the domain's word. A component drawing a stencil is a `Stencil` and takes a stencil. What ADR-0023 keeps out of the interface is the vault's types and its questions, not its vocabulary.

### The vocabulary is one page, grouped by context

Every term is in [`../glossary.md`](../glossary.md), under the context it belongs to. A word appearing in two contexts has an entry in each. A new term is added in the change that introduces it.

An entry says what a word means. It forbids another word only where that word means something else in the same context, and it never forbids the term the field itself uses.

## Consequences

- A name is looked up — first in the field, then in the glossary — and not settled by taste at each new file.
- A word carries a context with it. Reading a name means knowing which context the file sits in.
- Nothing checks a name against the glossary. A drifted word is found by a reader.
- A rename reaches the schema, the wire and the vault file format, and costs a migration there. Nothing has shipped, so it costs nothing else.

## Alternatives considered

**A word means one thing across the whole product.** Rejected: it makes every context compete for the same words, and the loser coins one. A thing everybody calls an icon was named a mark here because a card's mark had the word, and a reader now learns a private term for something they already knew.

**A dialect per boundary — a storage vocabulary, a wire vocabulary, an interface vocabulary.** Rejected: a word that cannot survive a boundary is the wrong word, and three vocabularies are three places for one concept to drift. A context is not a boundary: the domain, the storage and the wire all speak the vault's language, and it is a different subject, not a different layer, that begins a new context.

**Record the collisions and rename nothing.** Rejected: a list of known ambiguities that nobody acts on is a list that grows, and the reader goes on translating in their head.
