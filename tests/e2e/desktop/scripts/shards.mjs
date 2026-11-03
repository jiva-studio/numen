// The split of the spec files into shards. A shard is one Playwright process with one server and
// one vault; the files of a shard run one after the other, since the specs of one server share its
// settings, open tabs, trash and vault list.
//
// `shards.json` holds every spec file with the seconds it takes (the mean of the two engines), and
// `planShards` deals the files out by those seconds, longest first, to the shard with the least
// load. A spec file that `shards.json` does not list fails `checkShards`, so no file is left out
// of a run unnoticed. `scripts/measure-shards.mjs` writes the file from Playwright JSON reports.
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const shardsFile = path.join(root, 'shards.json')

/** The spec files under tests/, relative to the e2e folder, with `/` as separator, sorted. */
export const listSpecs = () => {
  const found = []
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name)
      if (entry.isDirectory()) walk(full)
      else if (entry.name.endsWith('.spec.ts')) {
        found.push(path.relative(root, full).split(path.sep).join('/'))
      }
    }
  }
  walk(path.join(root, 'tests'))
  return found.sort()
}

/** The seconds of every spec file in `shards.json`. */
export const readDurations = () => JSON.parse(fs.readFileSync(shardsFile, 'utf8'))

/** Writes the seconds of the spec files to `shards.json`, in path order. */
export const writeDurations = (durations) => {
  const sorted = Object.fromEntries(
    Object.entries(durations).sort(([a], [b]) => a.localeCompare(b)),
  )
  fs.writeFileSync(shardsFile, JSON.stringify(sorted, null, 2) + '\n')
}

/** The problems of `shards.json` against the spec files: files it lacks, and entries without a file. */
export const checkShards = () => {
  const durations = readDurations()
  const specs = new Set(listSpecs())
  return [
    ...[...specs]
      .filter((spec) => !(spec in durations))
      .map((spec) => `${spec} is in no shard: add it to shards.json (npm run shards:measure)`),
    ...Object.keys(durations)
      .filter((spec) => !specs.has(spec))
      .map((spec) => `shards.json lists ${spec}, which is not a spec file`),
  ]
}

/** The shard count when none is named: half the cores, at most four. */
export const getDefaultShardCount = () =>
  Math.max(1, Math.min(4, Math.floor(os.availableParallelism() / 2)))

/** Whether `spec` is one of `paths`: a path ending in `/` names a folder, any other a file. */
const isSelected = (spec, paths) =>
  paths.some((one) => (one.endsWith('/') ? spec.startsWith(one) : spec === one))

/** The seconds of the spec files of `paths`, or of every spec file when `paths` is undefined. */
export const readSelectedDurations = (paths) =>
  Object.fromEntries(
    Object.entries(readDurations()).filter(([spec]) => !paths || isSelected(spec, paths)),
  )

/** The paths NUMEN_E2E_PATHS names, separated by spaces, or undefined when it names none. */
export const readPathsFromEnv = () => {
  const paths = (process.env.NUMEN_E2E_PATHS ?? '').split(' ').filter(Boolean)
  return paths.length > 0 ? paths : undefined
}

/**
 * The spec files of each of `count` shards, out of the files of `paths` (every file when it is
 * undefined): longest first, each to the shard with the least load.
 */
export const planShards = (count, paths) => {
  const problems = checkShards()
  if (problems.length > 0) throw new Error(problems.join('\n'))
  const durations = readSelectedDurations(paths)
  const shards = Array.from({ length: count }, () => ({ load: 0, files: [] }))
  const byLength = Object.entries(durations).sort(([a, x], [b, y]) => y - x || a.localeCompare(b))
  for (const [spec, seconds] of byLength) {
    const lightest = shards.reduce((least, one) => (one.load < least.load ? one : least))
    lightest.load += seconds
    lightest.files.push(spec)
  }
  return shards.map((one) => one.files.sort())
}
