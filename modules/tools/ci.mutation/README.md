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

Neither tool writes to the checked-out source.

## The cache

Each target keeps one cache in the Actions cache, `mutation-<target>-run-<run id>`: the gomutants cache of a Go module, or the Stryker incremental report of a package. Every shard starts from it and measures again only the mutants whose code or covering tests changed, and uploads the cache it ends with as the artifact `cache-<shard>`.

After the shards, `cache.mjs` merges the target's cache. Each package or file is taken from the shard that holds it under the current shard count, and kept from the previous cache when that shard uploaded nothing. A shard that runs out of time uploads what it measured, and goes on from there the next night. A change in shard count moves files between shards and keeps their results.

- **gomutants** keeps entries for packages outside the ones it is given, as long as their files are unchanged, so each shard's cache holds the whole module; the merge takes a shard's entries for its own packages only. The merged cache keeps the identity gomutants checks before reusing anything (schema, module, tool version, tags, test flags, toolchain) and drops the coverage profile, which belongs to one shard's packages. An entry's test hash covers same-named tests in the other packages of the shard that wrote it, so after a change in shard count some entries miss and are measured again.
- **Stryker** numbers its tests anew in each report, so the merge renames every test by its file, position and name. A test is kept, and a mutant still names it, only where the test file's recorded source is the file's source now; a mutant whose killing test is dropped is measured again. Stryker reports every file its incremental report holds, and a shard's report keeps the files the shard mutated.

On Sunday the shards start without the cache and measure every mutant again, which catches what reuse misses: an edit in a package a Go mutant's package imports, or a change Stryker's coverage does not see. Their result is merged and saved as on any other night.

The Go shards read the build cache their module's own CI saves (`go-build-core-ubuntu-latest-`, `go-build-modules/apps/desktop-`, `go-build-modules/apps/mobile-`) and write none.

## The issues

Each target has one issue labelled `mutation`, titled `Mutation survivors: <target>`. Every night its body is rewritten with the current survivors and the count of the run before, and a comment names the survivors that are new. The body lists one survivor per line, with the count of the others on that line. The issue closes when no survivor is left and reopens when one comes back. A target with a shard that did not finish keeps its issue as it was.

A survivor is dealt with in one of two ways:

- a test that fails on it, which kills it the next night;
- an inline directive with the reason, when the mutant does not change behaviour. Stryker reads `// Stryker disable next-line <Mutator>: <reason>` (also `disable all` and `restore`); gomutants reads `// gomutants:disable <MUTATOR> reason="<reason>"` on the same line, the next line, a function doc comment, or as a regexp.

Interface copy is not mutated: each `words.ts` file opens with `// Stryker disable StringLiteral: interface copy`.

Test support is not mutated: files under `testing/`, `testsupport/` and `fixtures/` folders, `*_test.go`, `*.test.ts`, `*.stories.ts` and `testdata/`.

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
