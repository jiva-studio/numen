# Architecture decisions

Numbers are identity, not order. An ADR keeps its number forever, including when
it is superseded, so that every reference to it stays valid. Decisions arrive in
whatever order the work demands, which is rarely the order they are best read in.

**This page is the reading order.** The numbers on the files are not.

**Every ADR says what it applies to.** This repository holds several
applications, and a decision about one of them is not a decision about the
product. The header of each says which: the product as a whole, the vault
format, or a named application. Without that line a reader has to guess whether
a rule binds them, and guessing wrong in either direction is expensive.

To change an accepted decision, write a new ADR that supersedes it and mark the
old one `Superseded by ADR-NNNN`. Do not edit the old file into agreement with
the new one — the point of the record is that it shows what was believed and why.

**A decision is in the record once it is on the default branch.** Until then it
is its pull request's draft, and a draft is edited in place.

## Foundations

Read these first; everything else is derived from them.

- [ADR-0000 — Data is either artifact or cache](0000-artifact-vs-cache.md)
- [ADR-0001 — Files on disk are the source of truth](0001-files-are-the-source-of-truth.md)
- [ADR-0002 — SQLite is a cache, one database for all vaults](0002-sqlite-is-a-cache.md)
- [ADR-0024 — Ubiquitous language](0024-ubiquitous-language.md)

## Vaults and code

- [ADR-0013 — Vault identity, the service folder, and application state](0013-vault-identity-and-application-state.md)
- [ADR-0014 — One binary, hexagonal core in Go](0014-one-binary-hexagonal-core-in-go.md)
- [ADR-0015 — A schema change migrates the index instead of rebuilding it](0015-schema-changes-are-migrations.md)
- [ADR-0016 — What a scan stores about a note](0016-what-the-index-stores.md)
- [ADR-0017 — How the desktop application is tested](0017-how-the-desktop-application-is-tested.md)
- [ADR-0018 — A scan runs in the background, and there is one writer](0018-a-scan-runs-in-the-background.md)
- [ADR-0019 — Performance targets for indexing and search](0019-performance-targets.md)
- [ADR-0021 — The index measures itself after a scan](0021-the-index-measures-itself.md)
- [ADR-0022 — Notes are indexed in groups](0022-notes-are-indexed-in-groups.md)
- [ADR-0023 — The vault is watched](0023-the-vault-is-watched.md)
- [ADR-0029 — The vector index stays inside SQLite, and what that costs](0029-the-vector-index-and-what-it-costs.md)
- [ADR-0030 — The index-size budget, and where an exact scan ends](0030-index-size-and-approximate-search.md)
- [ADR-0031 — The agent this application starts, and what it may reach](0031-the-agent-this-application-starts.md)
- [ADR-0027 — The application writes to the vault](0027-the-application-writes-to-the-vault.md)
- [ADR-0032 — The window saves a note as it is typed, and stops at an unseen edit](0032-the-window-saves-a-note-as-it-is-typed.md)
- [ADR-0033 — One process, one writer, one lifetime](0033-one-process-one-writer-one-lifetime.md)

## The note file

- [ADR-0012 — A note is plain markdown any editor can open](0012-a-note-is-plain-markdown.md)
- Format specification: [note-format.md](../note-format.md)

## The interface

- [ADR-0020 — How an interface component is built](0020-how-an-interface-component-is-built.md)
- [ADR-0028 — The component library is shadcn-vue on Tailwind](0028-the-component-library-is-shadcn-vue.md)
- [ADR-0025 — A client talks to the core through a schema](0025-a-client-talks-to-the-core-through-a-schema.md)

## Agents

- [ADR-0026 — An agent reaches the vault through tools](0026-an-agent-reaches-the-vault-through-tools.md)

## Links and addressing

- [ADR-0003 — A link is one object carrying a role and an optional type](0003-a-link-carries-a-role.md)
- [ADR-0009 — Identifiers for notes and blocks](0009-identifiers.md)
- [ADR-0010 — Attachments are links](0010-attachments-are-links.md)
- [ADR-0011 — Links are written by name; `note://` is the auxiliary form](0011-links-are-written-by-name.md)

## Vault layout and sources

- [ADR-0006 — Sources: extracted text into the cache, unreproducible output into the vault](0006-sources.md)
- [ADR-0007 — Structural chunking, and how a passage is found](0007-structural-chunking-and-hybrid-search.md)
- [ADR-0034 — A chunk is identified by the text it holds](0034-a-chunk-is-identified-by-its-text.md)

Not yet written.

- ADR-0004 — A submodule mechanism for extension-produced artifacts ([#5](https://github.com/jiva-studio/numen/issues/5))

## Spaced repetition

Not yet written. Implementation is priority 2, but the formats are fixed early
because the note file and the index depend on them.

- ADR-0005 — Event sourcing for spaced repetition ([#6](https://github.com/jiva-studio/numen/issues/6))
- ADR-0008 — Card identity lives in the text ([#9](https://github.com/jiva-studio/numen/issues/9))
