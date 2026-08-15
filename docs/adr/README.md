# Architecture decisions

Numbers are identity, not order. An ADR keeps its number forever, including when
it is superseded, so that every reference to it stays valid. Decisions arrive in
whatever order the work demands, which is rarely the order they are best read in.

**This page is the reading order.** The numbers on the files are not.

To change an accepted decision, write a new ADR that supersedes it and mark the
old one `Superseded by ADR-NNNN`. Do not edit the old file into agreement with
the new one — the point of the record is that it shows what was believed and why.

## Foundations

Read these first; everything else is derived from them.

- [ADR-0000 — Data is either artifact or cache](0000-artifact-vs-cache.md)
- [ADR-0001 — Files on disk are the source of truth](0001-files-are-the-source-of-truth.md)
- [ADR-0002 — SQLite is a cache, one database for all vaults](0002-sqlite-is-a-cache.md)

## Vaults and code

- [ADR-0013 — Vault identity, the service folder, and application state](0013-vault-identity-and-application-state.md)
- [ADR-0014 — One binary, hexagonal core in Go](0014-one-binary-hexagonal-core-in-go.md)
- [ADR-0015 — The index is migrated, not rebuilt](0015-the-index-is-migrated.md)

## The note file

- [ADR-0012 — A note is plain markdown any editor can open](0012-a-note-is-plain-markdown.md)
- Format specification: [note-format.md](../note-format.md)

## Links and addressing

Not yet written.

- ADR-0003 — A link is one object with a role and an optional type ([#4](https://github.com/jiva-studio/numen/issues/4))
- ADR-0009 — Identifiers: two levels of addressing ([#10](https://github.com/jiva-studio/numen/issues/10))
- ADR-0010 — An attachment is a link, not a place on disk ([#11](https://github.com/jiva-studio/numen/issues/11))
- ADR-0011 — Links are written by name; `note://` is the auxiliary form ([#12](https://github.com/jiva-studio/numen/issues/12))

## Vault layout and sources

Not yet written.

- ADR-0004 — A submodule mechanism for extension-produced artifacts ([#5](https://github.com/jiva-studio/numen/issues/5))
- ADR-0006 — Sources: extracted text into the cache, unreproducible output into the vault ([#7](https://github.com/jiva-studio/numen/issues/7))
- ADR-0007 — Structural chunking; hybrid search ([#8](https://github.com/jiva-studio/numen/issues/8))

## Spaced repetition

Not yet written. Implementation is priority 2, but the formats are fixed early
because the note file and the index depend on them.

- ADR-0005 — Event sourcing for spaced repetition ([#6](https://github.com/jiva-studio/numen/issues/6))
- ADR-0008 — Card identity lives in the text ([#9](https://github.com/jiva-studio/numen/issues/9))
