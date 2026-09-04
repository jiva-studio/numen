# ADR-0046: One service to a subject

- **Status:** Accepted
- **Date:** 2026-09-04
- **Applies to:** `modules/libs/protocol`; `modules/libs/core` — `adapter/webui`, `adapter/flashcardsui`
- **Related:** ADR-0004, ADR-0005, ADR-0030, ADR-0045

## Context

A window asks about the vault it shows, the vaults the installation holds, the cards in one, the file a person configures the installation in, the window itself, and what a model has made from a file. Those are different subjects, and two windows run over the one schema (ADR-0030). A binary that mounts a service has to answer the whole of it, so what is on a service decides what a binary must be able to do.

## Decision

### A service is one subject, and every question about that subject is on it

- `VaultService` — the vault a window is showing, taken whole: what it is, what has changed in it, and where in it the person stands.
- `FileService` — the tree that vault is filed in: what a folder holds, what stands at a path, and moving one, removing one, making one.
- `NoteService` — a note of that vault: its prose, its headings, its neighbourhood, the addresses written in it, and every way of writing one.
- `SearchService` — what that vault holds that answers what a person typed: its names, and the passages of its text.
- `VaultsService` — the vaults the installation holds, and putting another one in front of the person.
- `CardsService` — the stencils and decks a vault is arranged into.
- `PresetsService` — the presets that schedule them, and the curve of one.
- `FlashcardsService` — a sitting: what is owed, what is asked, what was answered.
- `AssetService` — what a file of the vault is, for whatever opens it: a document's pages, a recording's length and where its bytes are played from, and where a run of a source's text sits on the page.
- `ArtifactService` — what has been made from a file: listing it, making one, taking one away, and reading and writing the words a recording was heard as.
- `SettingsService` — the file a person configures the installation in.
- `ThemeService` — what the window is dressed in.
- `WindowService` — one window: which vault it has in front of the person, what is being done behind it, and what has to land before it goes.
- `AgentService` — the conversation in the panel.

### A binary mounts a service whole

A window answers every call of a service it serves. A question one binary cannot answer therefore does not go on a service that binary needs: it goes on a service of its own, or it stays where it is. What is left unanswered is what a composition binds no use case to, which is a fact about that build and not about the shape of the wire.

The two calls that read a deck's preset are the case. The editor builds the use case that writes a preset and the review window does not, so folding the review window's call into `PresetsService` would put `MakePreset` in front of a binary that cannot answer it — or answer it unimplemented, which is the thing this split exists to take out. The two stay, and the whole of the duplication is one pair of messages.

The files of the vault are the other. The phone serves the vault's notes to a network, over a socket answering any origin at all, and must serve none of what is on the person's disk beside them: a book's pages, where a recording is played from, a model set running over either. That is why what a file *is* is `AssetService`, and why the phone declines it and `ArtifactService` whole. `FileService` is the tree and not the bytes — what a folder holds and where a file is filed — and the phone draws its own tree out of it, so it mounts that one.

### A window is a scope

Every call of `WindowService` names the window it is about, and one that names another window than the one answering is not answered. Two windows are open on one installation at once, and which vault each has in front of the person, what each is doing behind itself, and what it is owed before it can go, are the window's and not the installation's. That is why the list of vaults no longer says which one is being shown: the list is the same for both windows and the answer is not.

The review window is open on the installation rather than on any one vault — it says what is owed across all of them and names a vault in every call — so it answers `Showing` with none. That is the answer, not a stub: a window standing on nothing gives the same one.

### A type is shared once three services hold it

A type moves into the file the other services import only when three of them already use it. Two services holding one type is a coincidence; three is a shape. `Refusal`, `Fingerprint`, `Stretch` and `NoteType` stand there on those terms; everything else stays in the file of the service that answers with it, and the service that wants it imports that file. `Note` and `Heading` are `NoteService`'s and `SearchService` imports them; `SourceKind` and `Moved` are `FileService`'s, and `NoteService` and `SearchService` import them.

Under any looser rule the shared file admits whatever might be wanted twice and fills with types nothing in particular owns, and a type that arrives there early is one every service is written around afterwards.

### The service's name in front of a message is a cost of the flat package, not a reason to move it

Every file is the one proto package, so a message name is unique across the whole wire, and two services that both answer a `List` cannot both have a `ListRequest`. The lint's answer is to put the service's name in front of the second, and that is where the cost is paid. It is not paid by lifting the message somewhere shared.

## Consequences

- **A question is asked of the service that owns its subject**, and the binary that serves it can answer all of it.
- **A second window costs the services it needs and no more**, rather than a share of one service it must stub out the rest of.
- **A service is the unit of what a binary can do**, so a call added to one is a call every binary serving it must answer.
- **A handful of messages carry their service's name.** The number grows with the verbs two services share, and each is a name in one file and nothing else.
- **Two calls read a deck's preset.** Whichever is changed, the other is changed with it.

## Alternatives considered

**A proto package per service.** Rejected: `Refusal` and `Fingerprint` would then be imported across packages, and every generated client would carry a path per service where it now carries one. The prefixed names are the whole of what the flat package costs.

**A service per window.** Rejected: both windows ask about the agent, the theme and the window itself, so a service per window is those three written twice and drifting apart.

**Folding the two calls that read a deck's preset.** Deferred: it makes the review window answer for a use case it does not build. It is worth doing on the day that use case is built there, and not before.
