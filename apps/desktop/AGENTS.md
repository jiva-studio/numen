# Desktop Windows — Architecture and Conventions

The conventions shared across all languages in this repository are in `AGENTS.md` at
the root. The frontend sections apply to every `.ts` and `.vue` file.
This file defines rules for `apps/desktop/editor` and `apps/desktop/flashcards`. Where a piece of window code goes and how a slice is shaped is the skill [`nm-place-window`](../../.agents/skills/nm-place-window/SKILL.md); the Go side of `apps/desktop` follows [`nm-place-core`](../../.agents/skills/nm-place-core/SKILL.md).

## Layer hierarchy

Files in the desktop application belong to one of five architectural layers and may only import from lower layers.

| Layer      | Contains                                           |
| ---------- | -------------------------------------------------- |
| `shared`   | system types, paths, transport, base components    |
| `entities` | a business thing: note, deck, media, settings, tab |
| `features` | one user scenario, whole                           |
| `pages`    | one tab, whole                                     |
| `app`      | mounting, wiring, providers                        |

There is no `widgets` layer: a part that two pages use is a feature or an entity. A slice's segments are `ui`, `api`, `model`, `lib`, `config`. A slice imports no
sibling slice of its own layer, and there is no `@x` cross-import: the rule
`no-cross-slice-entry` of `tools/lint-layers` fails on any import of an `@x`
path. Code that two slices share is in a lower layer.

## What a `.vue` may not do

A `.vue` file uses no client, and a component receives only props. What talks to the core is the `api` segment of a slice, which binds the generated clients from `shared/api` and which the `model` of the same slice imports directly. A command is declared in the `model/commands.ts` of the feature that owns it; `entities/command` holds the `Command` type and, for now, the palette engine, which moves to `shared` in a later package. Lengths in styles: a `rem` or `em` literal on the property or a token, `px` only for a 1px hairline border. Business logic is a composable or a domain
model, never a component.

There is no store framework and no module-level mutable state.

## Calling the core

A window talks to the core through the generated Connect clients, and the response
is mapped field by field into the window's own type. A fault is converted to text by `formatErrorMessage` and an error code by `formatErrorCodeMessage`, both from `@numen/core-client`; nothing prints an error by stringifying it. A message for display is thrown as a `ReportedError`; any other error shows the generic text.

## Build the UI library before the window suites

A window imports `@numen/ui` from its **build**. Run `npm run build` in
`libs/ui/components` before either window's suite, or it fails on
`Cannot find module '@numen/ui'` — and a stale build is worse, because it
compiles and renders the old output.

## The PGO profile

`cmd/numen/default.pgo` is the CPU profile `go build` reads through `-pgo=auto`. `make app_desktop_pgo` rebuilds it from the benchmarks of the scan, search, flashcards and editor paths; run it on an idle machine, and again when those paths change enough to change which functions are hot. `go version -m` on the built binary shows `-pgo=` when the profile was used. The refresh ends by rewriting every file name in the profile to a repository-relative, `$GOROOT` or `$GOMODCACHE` path and fails if a home directory or drive path remains; `make app_desktop_pgo_check` runs that check on the committed file.
