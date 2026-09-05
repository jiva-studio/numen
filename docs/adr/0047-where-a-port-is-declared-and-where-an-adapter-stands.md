# ADR-0047: Where a port is declared, and where an adapter stands

- **Status:** Accepted
- **Date:** 2026-09-05
- **Applies to:** `modules/libs/core` — `port`, `adapter`, `internal/adapter`, `container`
- **Amends:** ADR-0004
- **Related:** ADR-0004, ADR-0021, ADR-0022, ADR-0046

## Context

The core declares thirty-six interfaces in one folder, and whether a given one belongs there has been settled a case at a time: by how many callers it has, by which way the call goes through it, by which folder read better on the day. Each answer stood on its own and no two were the same rule, so the next one is argued from the start again.

Two questions are the shape of it. A one-method interface a single use case asks for — is it a port, or is one caller too few? And an adapter that serves the schema while sitting under `internal/` — is it in the wrong folder, or is the folder about something else?

## Decision

### A port is a conversation, and a conversation is not counted in callers

A port is a purposeful conversation between the core and something outside it. `VaultWatcher` is the conversation *tell me when the folder changed*; that one use case is the only one who wants telling is a fact about this application, not about the conversation. An interface asked for once and an interface asked for six times are the same kind of thing.

**The number of consumers decides nothing.** It is not a criterion in the literature this architecture is taken from, and the two extremes it would push towards are both named there as wrong: a port for every use case, and one port to a side.

`IndexMaintenance`, `VaultWatcher` and `VectorQueries` are ports. So are the thirty-three others.

### What has to see it decides where it is declared

An interface is declared where everything that uses values of it can see it, and no wider.

The composition root uses values of every port it binds: it builds the adapter and hands it over. It is therefore a consumer of all of them, and the folder holding what that one consumer must see is `port`. This is the whole of the rule, and it is the rule ADR-0004 already states — a port an adapter is bound to in `container` is declared in `port`, and a one-method interface a single use case needs, that nothing binds, is declared beside that use case.

The four whose only callers are adapters are decided by the same sentence. `Window` and `FolderDialog` are asked for by the window's adapter and the tool endpoint's; `Agent` by the window's and the review window's (ADR-0022); `IndexProgress` by the window's and the composition root. Two adapters must see one interface, and an adapter takes no other adapter, so the only place both can see it is the core. They are ports because more than one adapter has to see them, not because of which side of the hexagon they sit on.

### A port is named in the core's own language, whatever that costs the caller

A conversation is held in the words of what it is about. `Documents` speaks of a highlight's box and a part's start, `Recogniser` of a block of a page, `Proofreader` of a batch — and each of those words belongs to a package of the core. A caller that wants only `VaultWatcher` is compiled from all of them, and that is the price.

The alternative is a second vocabulary declared at the boundary and translated on both sides of it, which buys a smaller build graph with a duplicate domain. The words stay where the domain keeps them.

### An adapter stands where what composes it can reach it

`internal/` is what nothing outside this module composes. It is Go's visibility and nothing else, and it says who may build the adapter — never which way a call goes through it.

**Which way an adapter faces has nothing to do with where it stands.** The asymmetry the architecture exploits is between the inside and the outside of the application, not between its left and its right; a driving adapter and a driven one are both outside, and both are held or public according to who builds them.

`internal/adapter/theme` is a driving adapter: it answers the schema's theme calls, and it watches a folder to answer one of them. Nothing outside the core composes it — the two windows mount it, and an application reaches it through `container` — so it stands under `internal/`, and it would stand there if it were driven, and it would stand in `adapter/` if an application built one for itself.

### The two facts are recorded apart, and both are checked

Direction is read off what an adapter does. An adapter that takes the generated messages and answers with them is a driving adapter, wherever it stands, and the list of driving adapters in the layer guard says so. It is read off the messages and not off the handler, because an adapter never names the interface it answers to.

`container/layers_test.go` holds both checks:

- **Every port is asked for somewhere else.** An interface left in `port` after its last caller went is indirection standing on its own, and the composition root goes on binding an adapter to it.
- **Every adapter that speaks the schema is named driving.** A driving adapter that has been filed as driven is refused a scenario on the day it needs one, and refused for the wrong reason.

### Where this comes from

- Alistair Cockburn, [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/) — "A port identifies a purposeful conversation"; "every use case could be given its own port, producing hundreds of ports … Alternatively, one could imagine merging all primary ports and all secondary ports so there are only two ports … Neither extreme appears optimal"; "The asymmetry to exploit is not that between *left* and *right* sides of the application but between *inside* and *outside*".
- Robert C. Martin, [The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) — source code dependencies point inwards only, and what crosses a boundary is an isolated, simple data structure.
- Eric Evans, *Domain-Driven Design* — a repository is declared in the domain layer and implemented in infrastructure, one to an aggregate root.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments#interfaces) — "Go interfaces generally belong in the package that uses values of the interface type, not the package that implements those values", and "Do not define interfaces before they are used".

## Consequences

- **A port is argued from the conversation it holds**, and a review that says "only one caller" is answered by this record rather than by a fresh argument.
- **A caller that wants one port is compiled from the language every port is named in.** The build graph is wider than the call graph, and the words are in one place.
- **A driving adapter can stand under `internal/`.** Reading the tree does not say which way an adapter faces, and the layer guard is where that is written down.
- **A port outliving its last caller fails the build.** So does a driving adapter filed as driven.

## Alternatives considered

**A port for each caller, or a folder of ports for each aggregate.** Rejected: it is the extreme Cockburn names, the interfaces are the same interfaces, and the composition root then looks in a place per aggregate for what it binds in one.

**Deciding by the layer of the caller: an interface only an adapter asks for is not a port.** Rejected: two adapters would then share it by one naming the other, which is the thing the layer rule exists to refuse, and `port.Agent` is a port by ADR-0022.

**Declaring the words a port is named in at the boundary, so `port` is built from nothing.** Rejected: a highlight's box and a block of a page are the domain's, and a second set of them is two vocabularies to keep in step for a shorter import list.

**Moving `theme` to `adapter/`, on the ground that it is driving.** Rejected: nothing outside the core composes it, so making it public puts an adapter on the core's surface that no application names.
