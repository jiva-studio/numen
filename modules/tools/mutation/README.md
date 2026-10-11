# Mutation testing

A mutant is a small change to the code — a condition inverted, a statement removed, a constant shifted. A mutant the tests still pass is a line whose behaviour no test defends.

`.github/workflows/mutation.yml` mutates every target each night, split into shards that run side by side:

| Target | Tool | What runs per mutant |
|---|---|---|
| `core` (`modules/libs/core`) | [gomutants](https://github.com/szhekpisov/gomutants), through `go test -overlay` | the package tests that cover the mutated line |
| `ui`, `editor`, `flashcards` | [Stryker](https://stryker-mutator.io) with its Vitest runner | the unit tests that cover the mutated code, from `vitest.mutation.config.ts` |

Neither tool writes to the checked-out source. Each shard keeps its results in the Actions cache, and the next night measures again only the mutants whose code or covering tests changed. A shard that runs out of time saves what it measured and goes on from there the next night.

## The issues

Each target has one issue labelled `mutation`, titled `Mutation survivors: <target>`. Every night its body is rewritten with the current survivors, and a comment names the ones that are new. The issue closes when no survivor is left and reopens when one comes back. A target with a shard that did not finish keeps its issue as it was.

A survivor is dealt with in one of two ways:

- a test that fails on it, which kills it the next night;
- a waiver, when the mutant does not change behaviour: its key — the grey digits at the end of its line in the issue — goes into `waivers.json` with the reason.

```json
{ "waived": [{ "target": "core", "key": "16d8d7435c", "why": "256 and 257 both exceed every table size the tests build" }] }
```

The key holds the file, the mutator, the replaced text and the replacement, so it survives edits elsewhere in the file.

## Running a shard by hand

```bash
node modules/tools/mutation/mutate.mjs core --shard 3/12   # writes modules/.mutation/reports/core-3-of-12.json
node modules/tools/mutation/issue.mjs core --dry-run       # prints the issue body from the reports present
```

`NUMEN_MUTATION_WORKERS` sets the parallelism; it defaults to every core. A `ts` target needs `@numen/ui` built first.

## The Vitest runner patch

`@stryker-mutator/vitest-runner` 10.0.0 names a test with its suite chain joined by spaces, and Vitest 5 matches names joined by ` > `, so the runner selects no test per mutant and every mutant survives ([stryker-js#6210](https://github.com/stryker-mutator/stryker-js/issues/6210)). `patch.mjs` rewrites the join before each run, and refuses any other version of the runner. A run in which several mutants survive and none is killed fails as a broken runner.
