# The windows — what holds here

The conventions every language in this repository shares are in `AGENTS.md` at
the root, and the frontend sections of it hold over every `.ts` and `.vue` in
the repository. This file is the two windows' own, and it holds over
`modules/apps/desktop/editor` and `modules/apps/desktop/flashcards`.

## Where a file stands

A file of a window stands on one of six layers, and reaches only what stands
below it.

| Layer | Holds |
|---|---|
| `shared` | system types, paths, transport, base components |
| `entities` | a business thing: note, deck, media, settings, tab |
| `features` | one user scenario, whole |
| `widgets` | a composite block a page puts together |
| `pages` | one tab, whole |
| `app` | mounting, wiring, providers |

A slice's segments are `ui`, `api`, `model`, `lib`, `config`. A slice reaches no
sibling slice of its own layer; where two are bound by the domain, the one asked
of declares a public API under `@x`, named for the slice it is for.

## What a `.vue` may not do

A `.vue` file reaches no client. What talks to the core is a port in `app/`, and
what a component is given is props. Business logic is a composable or a domain
model, never a component.

There is no store framework and no module-level mutable state.

## Reaching the core

A window talks to the core through the generated Connect clients, and the answer
is mapped field by field into the window's own type. A refusal is put into words
by `refusalWords` or `troubleWords` from `@numen/wire`; nothing prints an error
by stringifying it.

## The one gotcha that costs an afternoon

A window reaches `@numen/ui` through its **build**. Run `npm run build` in
`modules/libs/ui` before either window's suite, or it fails on
`Cannot find module '@numen/ui'` — and a stale build is worse, because it
compiles and draws the wrong thing.

## The PGO profile

`cmd/numen/default.pgo` is the CPU profile `go build` reads through `-pgo=auto`. `make app_desktop_pgo` rebuilds it from the benchmarks of the scan, search, flashcards and editor paths; run it on an idle machine, and again when those paths change enough to move what is hot. `go version -m` on the built binary shows `-pgo=` when the profile was used. The refresh ends by rewriting every file name in the profile to a repository-relative, `$GOROOT` or `$GOMODCACHE` path and fails if a home directory or drive path remains; `make app_desktop_pgo_check` runs that check on the committed file.
