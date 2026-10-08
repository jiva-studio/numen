# An import says which layer it reaches

- **Status:** Accepted
- **Date:** 2026-09-12
- **Applies to:** `modules/apps/desktop/editor`, `modules/libs/ui`
- **Related:** [A file of the windows stands on a layer](0042-a-file-of-the-windows-stands-on-a-layer.md)

## Context

A path out of a folder said how many folders to climb and nothing about what it arrived at. `../../entities/note` and `../../features/command-palette` read the same, though one goes down the layers and the other goes up; `'../../shared/../features/command-palette/lists'` went up through the root and back down, and no reader saw it. Every file moved took a handful of these with it.

## Decision

`@/` is the window's own `src/`, declared in its `tsconfig.json`, its `vite.config.ts` and its `vitest.config.ts`, all three, so the build, the type check and the tests resolve one thing.

An import that leaves its own folder is written through it: `@/entities/note`, not `../../entities/note`. An import that stays inside the folder keeps its relative form — `./tab`, `./types` — because a slice moved whole should not take a rewrite with it.

Two paths are outside this: a fixture the schema package holds, which is not under `src/` and which `@/` cannot name, and a file nested deeper inside its own slice reaching the top of that slice.

### What checks it is not the boundary check

A cruise judges where an import lands. `../../entities/note` and `@/entities/note` land on the same file, so it has nothing to say about which was written. The rule reading the text is `modules/tools/lint/reaching.mjs`, and it refuses a path climbing out of its own slice by counting folders.

## Consequences

Which layer an import reaches is read off the import. The boundary check resolves `@/` through the module's own `tsconfig`, and the `nothing-unresolved` rule it already carries means an alias declared in one place and not another fails loudly rather than quietly widening what the walk cannot see.

A file moved between folders no longer drags its neighbours' import paths behind it.
