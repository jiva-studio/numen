#!/usr/bin/env node
/**
 * One cache per target, merged from what its shards wrote.
 *
 *   node cache.mjs <target> <dir>
 *
 * Every shard reads the target's cache from `.mutation/cache/` and
 * writes it back with its own results. `<dir>` holds one folder per shard,
 * `cache-<shard name>`, with the file that shard wrote. Each file or package
 * is taken from the shard that holds it under the current count, and from the
 * previous cache when that shard left nothing.
 */
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { listShards } from './shards.mjs'
import { OUTPUT, getShardOf, getTarget, root } from './targets.mjs'

/** The name of a target's cache file, as each tool writes it. */
export const getCacheFile = (target) =>
  `${target.name}.${target.kind === 'go' ? 'gomutants' : 'stryker'}.json`

/* ---------------------------------- Go ------------------------------------ */

/** The fields gomutants compares before it reuses any entry of a cache. */
const GO_IDENTITY = [
  'schema_version',
  'go_module',
  'tool_version',
  'build_tags',
  'test_flags',
  'go_toolchain',
]

const getGoIdentity = (cache) => JSON.stringify(GO_IDENTITY.map((field) => cache[field] ?? ''))

/** The shard of `count` holding the package of a module-relative Go file. */
export const getGoShardOf = (relFile, count) => {
  const dir = dirname(relFile)
  return getShardOf(dir === '.' ? '.' : `./${dir}`, count)
}

/**
 * The gomutants cache of a target: each shard's entries for its own packages,
 * and the previous entries for the packages of shards that wrote nothing.
 * `outputs` are `{ index, cache }`. The coverage profile belongs to one
 * shard's package set and is left out.
 */
export const mergeGoCaches = (count, previous, outputs) => {
  const newest = outputs.at(-1)?.cache ?? previous
  if (!newest) return null
  const identity = getGoIdentity(newest)
  const current = outputs.filter((one) => getGoIdentity(one.cache) === identity)
  const ran = new Set(current.map((one) => one.index))
  const carried =
    previous && getGoIdentity(previous) === identity
      ? (previous.entries ?? []).filter((entry) => !ran.has(getGoShardOf(entry.rel_file, count)))
      : []
  const fresh = current.flatMap(({ index, cache }) =>
    (cache.entries ?? []).filter((entry) => getGoShardOf(entry.rel_file, count) === index),
  )
  const { coverage_key: _key, coverage_profile: _profile, entries: _entries, ...meta } = newest
  return { ...meta, entries: [...carried, ...fresh] }
}

/* ------------------------------- TypeScript ------------------------------- */

/** A test's identity across reports, as Stryker's incremental differ forms it. */
const getTestKey = (file, test) => {
  const start = test.location?.start ?? { line: 0, column: 0 }
  return `${file}@${start.line}:${start.column}\n${test.name}`
}

/**
 * The Stryker incremental report of a target. Files come from the shard that
 * holds them, or from the previous report when that shard wrote nothing. Test
 * ids are numbered anew; a test is kept, and named by a mutant, only where
 * its file's recorded source equals `readSource(file)`, the source now.
 */
export const mergeStrykerReports = (count, previous, outputs, readSource) => {
  const newest = outputs.at(-1)?.report ?? previous
  if (!newest) return null
  const ran = new Set(outputs.map((one) => one.index))
  const sources = [
    ...(previous ? [{ report: previous, isOwner: (index) => !ran.has(index) }] : []),
    ...outputs.map(({ index, report }) => ({ report, isOwner: (owner) => owner === index })),
  ]

  const testFiles = {}
  const ids = new Map()
  const keysBySource = sources.map(({ report }) => {
    const keys = new Map()
    for (const [file, testFile] of Object.entries(report.testFiles ?? {})) {
      const now = readSource(file)
      if (now === undefined || testFile.source !== now) continue
      testFiles[file] ??= { source: now, tests: [] }
      for (const test of testFile.tests) {
        const key = getTestKey(file, test)
        keys.set(test.id, key)
        if (ids.has(key)) continue
        ids.set(key, String(ids.size))
        testFiles[file].tests.push({ ...test, id: ids.get(key) })
      }
    }
    return keys
  })

  const files = {}
  sources.forEach(({ report, isOwner }, at) => {
    const remap = (testIds) =>
      testIds?.map((id) => ids.get(keysBySource[at].get(id))).filter((id) => id !== undefined)
    for (const [file, result] of Object.entries(report.files ?? {})) {
      if (!isOwner(getShardOf(file, count))) continue
      files[file] = {
        ...result,
        mutants: result.mutants.map((mutant) => ({
          ...mutant,
          killedBy: remap(mutant.killedBy),
          coveredBy: remap(mutant.coveredBy),
        })),
      }
    }
  })
  return { ...newest, files, testFiles }
}

/** A Stryker report reduced to the files a run mutated. */
export const keepFiles = (report, mutated) => {
  const kept = new Set(mutated)
  return {
    ...report,
    files: Object.fromEntries(
      Object.entries(report.files ?? {}).filter(([file]) => kept.has(file)),
    ),
  }
}

/* ---------------------------------- Main ---------------------------------- */

const readJson = (path) => (existsSync(path) ? JSON.parse(readFileSync(path, 'utf8')) : null)

const readSource = (file) => {
  const path = join(root, file)
  return existsSync(path) ? readFileSync(path, 'utf8') : undefined
}

const main = () => {
  const [name, dir] = process.argv.slice(2)
  if (!name || !dir) throw new Error('usage: cache.mjs <target> <dir>')
  const target = getTarget(name)
  const file = getCacheFile(target)
  const path = join(root, OUTPUT, 'cache', file)
  const previous = readJson(path)
  const outputs = listShards(target.name)
    .map((shard) => ({
      index: Number(shard.shard.split('/')[0]),
      data: readJson(join(dir, `cache-${shard.name}`, file)),
    }))
    .filter((one) => one.data !== null)
  const merged =
    target.kind === 'go'
      ? mergeGoCaches(
          target.shards,
          previous,
          outputs.map(({ index, data }) => ({ index, cache: data })),
        )
      : mergeStrykerReports(
          target.shards,
          previous,
          outputs.map(({ index, data }) => ({ index, report: data })),
          readSource,
        )
  console.log(
    `${target.name}: ${outputs.length} of ${target.shards} shard caches merged${previous ? ' over the previous cache' : ''}`,
  )
  if (merged) writeFileSync(path, JSON.stringify(merged))
}

if (process.argv[1] === new URL(import.meta.url).pathname) main()
