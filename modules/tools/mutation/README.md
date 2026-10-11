# Mutation testing

A mutant is a small change to the code: a condition inverted, a statement removed, a constant shifted. A mutant the tests still pass is a line whose behaviour no test defends.

`.github/workflows/mutation.yml` mutates every target each night, split into shards that run side by side.

## Targets

Targets are found in the tracked tree, and a new module or package is picked up the night after it lands:

- **Go:** every Go module under `modules/` with a `_test.go` of its own. [gomutants](https://github.com/szhekpisov/gomutants) applies each mutant through `go test -overlay`, runs the tests that cover the mutated line, and compiles each survivor to drop the ones whose machine code is unchanged.
- **TypeScript:** every workspace package with a `vitest.config.ts` and source under `src/`. [Stryker](https://stryker-mutator.io) runs with its Vitest runner against the package's projects that do not run in a browser, through a config written at run time into `modules/.mutation/vitest/`. Test data the tests import from outside the package is found from their import paths and kept in Stryker's sandbox.

A target is named by its path under `modules/` with dashes (`libs-core`, `apps-desktop-editor`), with `-go` or `-ts` appended where one folder holds both. It takes one shard per 6000 lines of Go or 4500 lines of TypeScript, up to 16. A package or file is placed in a shard by a hash of its path, so adding one moves no other.

```bash
node modules/tools/mutation/shards.mjs          # every target and its shards
```

Neither tool writes to the checked-out source. Each shard keeps its results in the Actions cache, and the next night measures again only the mutants whose code or covering tests changed. A shard that runs out of time saves what it measured and goes on from there the next night.

## The issues

Each target has one issue labelled `mutation`, titled `Mutation survivors: <target>`. Every night its body is rewritten with the current survivors and the count of the run before, and a comment names the survivors that are new. The body lists one survivor per line, with the count of the others on that line. The issue closes when no survivor is left and reopens when one comes back. A target with a shard that did not finish keeps its issue as it was.

A survivor is dealt with in one of two ways:

- a test that fails on it, which kills it the next night;
- a waiver, when the mutant does not change behaviour: its key, the grey digits at the end of its line in the issue, goes into `waivers.json` with the reason.

```json
{
  "waived": [
    {
      "target": "libs-core",
      "key": "16d8d7435c",
      "why": "256 and 257 both exceed every table the tests build"
    }
  ]
}
```

The key holds the file, the mutator, the replaced text and the replacement, so it survives edits elsewhere in the file.

## Checking one file

```bash
node modules/tools/mutation/mutate.mjs --file modules/apps/desktop/flashcards/src/app/usePanels.ts
```

mutates one source file with the target that holds it, uses no cache, and prints each mutant its tests do not notice. It is the check after writing tests against the survivors an issue lists.

## Running a shard by hand

```bash
node modules/tools/mutation/mutate.mjs libs-core --shard 3/12   # writes modules/.mutation/reports/libs-core-3-of-12.json
node modules/tools/mutation/issue.mjs libs-core --dry-run       # prints the issue body from the reports present
```

`NUMEN_MUTATION_WORKERS` sets the parallelism; it defaults to every core. A TypeScript target needs `@numen/ui` built first.

## The Vitest runner patch

`@stryker-mutator/vitest-runner` 10.0.0 names a test with its suite chain joined by single spaces, and Vitest 5 matches the chain joined by a spaced greater-than sign, so the runner selects no test per mutant and every mutant survives ([stryker-js#6210](https://github.com/stryker-mutator/stryker-js/issues/6210)). `patch.mjs` rewrites the join before each run and refuses any other version of the runner.
