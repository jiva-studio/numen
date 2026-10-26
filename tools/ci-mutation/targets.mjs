/**
 * What is under mutation testing, found in the tracked tree.
 *
 * A `go` target is every tracked Go module with a test of its own,
 * run with gomutants. A `ts` target is every workspace package with a
 * `vitest.config.ts` and source under `src/`, run with Stryker against its
 * projects that do not run in a browser. Each target is split into shards by
 * the size of its source.
 */
import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { dirname, join, normalize } from 'node:path'
import { fileURLToPath } from 'node:url'

export const root = join(dirname(fileURLToPath(import.meta.url)), '../..')

/** Where runs write their reports, caches and generated configs, relative to the repository. */
export const OUTPUT = '.mutation'

export const GOMUTANTS = 'github.com/szhekpisov/gomutants@v0.6.1'

/** Lines of source one shard takes, by kind, and the most shards a target is split into. */
export const SHARD_LINES = { go: 6000, ts: 4500 }
export const MAX_SHARDS = 16

/** Whether a path lies in a folder of helpers the tests are built with. */
export const isTestSupport = (path) => /(^|\/)(testing|testsupport|fixtures)\//.test(path)

/** Whether a repository path is source a `ts` target mutates. */
export const isMutatedSource = (path) =>
  /\.ts$/.test(path) &&
  !isTestSupport(path) &&
  !/\.(test|stories|d)\.ts$/.test(path) &&
  !/(^|\/)(index|types|main)\.ts$/.test(path) &&
  /\/src\//.test(path)

/** Whether a repository path is Go source gomutants mutates. */
export const isMutatedGo = (path) =>
  /\.go$/.test(path) &&
  !/_test\.go$/.test(path) &&
  !/(^|\/)testdata\//.test(path) &&
  !isTestSupport(path)

const hash = (text) => {
  let value = 2166136261
  for (const char of text) value = Math.imul(value ^ char.codePointAt(0), 16777619) >>> 0
  return value
}

/**
 * The shard of `count`, numbered from 1, that holds an item. It depends on the
 * item's name and the count only, so adding an item moves no other.
 */
export const getShardOf = (item, count) => (hash(item) % count) + 1

/** The items of shard `index` of `count`. */
export const pickShard = (items, index, count) =>
  items.filter((item) => getShardOf(item, count) === index)

export const getShardCount = (kind, lines) =>
  Math.min(MAX_SHARDS, Math.max(1, Math.ceil(lines / SHARD_LINES[kind])))

/**
 * Names for targets found at these folders: the repository path with
 * dashes, and the kind appended where one folder holds both.
 */
export const nameTargets = (found) => {
  const kinds = new Map()
  for (const one of found) kinds.set(one.at, (kinds.get(one.at) ?? 0) + 1)
  return found.map((one) => {
    const base = one.at.replaceAll('/', '-')
    return { ...one, name: kinds.get(one.at) > 1 ? `${base}-${one.kind}` : base }
  })
}

/**
 * Folders outside `at` that the tests read, from the relative specifiers in
 * their source: `../../protocol/testdata/faces.json` from a test in `at`
 * keeps `…/protocol/testdata`.
 */
export const findTestdata = (at, tests) => {
  const kept = new Set()
  for (const { path, source } of tests) {
    for (const [, specifier] of source.matchAll(/['"](\.\.?\/[^'"?]+)(?:\?[^'"]*)?['"]/g)) {
      const resolved = normalize(join(dirname(path), specifier))
      if (resolved.startsWith(`${at}/`) || resolved.startsWith('..')) continue
      kept.add(dirname(resolved))
    }
  }
  const sorted = [...kept].sort((a, b) => a.localeCompare(b))
  return sorted.filter(
    (one) => !sorted.some((other) => other !== one && one.startsWith(`${other}/`)),
  )
}

/* -------------------------------- Discovery ------------------------------- */

const listTracked = () =>
  execFileSync('git', ['ls-files'], {
    cwd: root,
    encoding: 'utf8',
    maxBuffer: 64 * 1024 * 1024,
  })
    .split('\n')
    .filter(Boolean)

const countLines = (paths) =>
  paths.reduce((sum, path) => sum + readFileSync(join(root, path), 'utf8').split('\n').length, 0)

const isUnder = (path, at) => path.startsWith(`${at}/`)

const findGoTargets = (tracked) => {
  const modules = tracked.filter((path) => path.endsWith('/go.mod')).map(dirname)
  const getOwner = (path) =>
    modules.filter((at) => isUnder(path, at)).sort((a, b) => b.length - a.length)[0]
  return modules
    .map((at) => {
      const files = tracked.filter((path) => path.endsWith('.go') && getOwner(path) === at)
      const sources = files.filter(isMutatedGo)
      return {
        kind: 'go',
        at,
        lines: countLines(sources),
        testdata: [],
        hasTests: files.some((path) => path.endsWith('_test.go')),
      }
    })
    .filter((one) => one.hasTests && one.lines > 0)
    .map(({ hasTests: _, ...one }) => one)
}

const readWorkspaces = () => {
  const { workspaces } = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'))
  return workspaces.map(
    (glob) => new RegExp(`^${glob.replaceAll('.', '\\.').replaceAll('*', '[^/]+')}$`),
  )
}

const findTsTargets = (tracked) => {
  const workspaces = readWorkspaces()
  return tracked
    .filter((path) => path.endsWith('/package.json'))
    .map(dirname)
    .filter((at) => workspaces.some((pattern) => pattern.test(at)))
    .filter((at) => tracked.includes(`${at}/vitest.config.ts`))
    .map((at) => {
      const sources = tracked.filter((path) => isUnder(path, `${at}/src`) && isMutatedSource(path))
      const tests = tracked
        .filter((path) => isUnder(path, at) && /\.test\.ts$/.test(path))
        .map((path) => ({ path, source: readFileSync(join(root, path), 'utf8') }))
      return { kind: 'ts', at, lines: countLines(sources), testdata: findTestdata(at, tests) }
    })
    .filter((one) => one.lines > 0)
}

let found = null

/** Every target, with `name`, `kind`, `at`, `shards` and the `testdata` folders its tests read. */
export const getTargets = () => {
  if (found) return found
  const tracked = listTracked()
  found = nameTargets([...findGoTargets(tracked), ...findTsTargets(tracked)])
    .map(({ lines, ...one }) => ({ ...one, shards: getShardCount(one.kind, lines) }))
    .sort((a, b) => a.name.localeCompare(b.name))
  return found
}

export const getTarget = (name) => {
  const targets = getTargets()
  const target = targets.find((one) => one.name === name)
  if (!target) {
    throw new Error(
      `no mutation target called ${name}; the targets are ${targets.map((one) => one.name).join(', ')}`,
    )
  }
  return target
}

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
