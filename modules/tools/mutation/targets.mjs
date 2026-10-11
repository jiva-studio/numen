/**
 * What is under mutation testing, and how each target is run.
 *
 * A `go` target is a module run with gomutants, which applies each mutant
 * through `go test -overlay` and leaves the tree as it found it. A `ts` target
 * is a workspace package run with Stryker and its Vitest runner, against the
 * package's `vitest.mutation.config.ts`: the unit project alone, since a story
 * drawn in a browser per mutant does not fit in a night.
 */
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

export const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')

/** Where runs write their reports and caches, relative to the repository. */
export const OUTPUT = 'modules/.mutation'

export const GOMUTANTS = 'github.com/szhekpisov/gomutants@v0.6.1'

/**
 * Each target by name, with:
 *
 * - `kind`, `go` or `ts`;
 * - `at`, the module or package folder, relative to the repository;
 * - `shards`, how many jobs a full run is split into;
 * - `testdata`, folders outside `at` its tests read, relative to the repository.
 */
export const targets = [
  { name: 'core', kind: 'go', at: 'modules/libs/core', shards: 12 },
  {
    name: 'ui',
    kind: 'ts',
    at: 'modules/libs/ui',
    shards: 4,
    testdata: ['modules/libs/protocol/testdata'],
  },
  {
    name: 'editor',
    kind: 'ts',
    at: 'modules/apps/desktop/editor',
    shards: 4,
    testdata: ['modules/libs/protocol/testdata'],
  },
  {
    name: 'flashcards',
    kind: 'ts',
    at: 'modules/apps/desktop/flashcards',
    shards: 1,
    testdata: ['modules/libs/protocol/testdata'],
  },
]

export const getTarget = (name) => {
  const found = targets.find((one) => one.name === name)
  if (!found) {
    throw new Error(
      `no mutation target called ${name}; the targets are ${targets.map((one) => one.name).join(', ')}`,
    )
  }
  return found
}

/** Whether a repository path is source a `ts` target mutates. */
export const isMutatedSource = (path) =>
  /\.ts$/.test(path) &&
  !/\.(test|stories|d)\.ts$/.test(path) &&
  !/(^|\/)(index|types|main)\.ts$/.test(path) &&
  /\/src\//.test(path)

/** Whether a repository path is Go source gomutants mutates. */
export const isMutatedGo = (path) =>
  /\.go$/.test(path) && !/_test\.go$/.test(path) && !/(^|\/)testdata\//.test(path)

/** The items whose position in the list falls in shard `index` of `count`, numbered from 1. */
export const pickShard = (items, index, count) => items.filter((_, at) => at % count === index - 1)

/**
 * Gitignore patterns that drop everything from the sandbox except `kept`,
 * each kept folder re-included level by level from the root.
 */
export const keepOnly = (kept) => {
  const patterns = ['/*']
  const seen = new Set()
  for (const path of kept) {
    const parts = path.split('/')
    for (let depth = 1; depth <= parts.length; depth++) {
      const prefix = parts.slice(0, depth).join('/')
      if (seen.has(prefix)) continue
      seen.add(prefix)
      patterns.push(`!/${prefix}`)
      if (depth < parts.length) patterns.push(`/${prefix}/*`)
    }
  }
  return patterns
}
