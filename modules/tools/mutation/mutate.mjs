#!/usr/bin/env node
/**
 * Runs mutation testing on one shard of a target and writes its report in the
 * Stryker schema to `modules/.mutation/reports/`.
 *
 *   node mutate.mjs <target> [--shard 2/8]
 *
 * Results of the previous run of the same shard, kept in `modules/.mutation/cache/`,
 * are reused wherever the code and its covering tests are unchanged.
 */
import { execFileSync, spawnSync } from 'node:child_process'
import { mkdirSync, readdirSync, rmSync, writeFileSync } from 'node:fs'
import { availableParallelism } from 'node:os'
import { dirname, join, relative } from 'node:path'
import { patchVitestRunner } from './patch.mjs'
import {
  GOMUTANTS,
  OUTPUT,
  getTarget,
  isMutatedGo,
  isMutatedSource,
  pickShard,
  root,
} from './targets.mjs'

const MODULES = join(root, 'modules')

const parseArgs = (argv) => {
  const [name, ...rest] = argv
  if (!name) throw new Error('usage: mutate.mjs <target> [--shard k/n]')
  const options = { target: getTarget(name), shard: [1, 1] }
  for (let at = 0; at < rest.length; at += 2) {
    if (rest[at] === '--shard') options.shard = rest[at + 1].split('/').map(Number)
    else throw new Error(`unknown option ${rest[at]}`)
  }
  const [index, count] = options.shard
  if (!(index >= 1 && index <= count)) throw new Error(`no shard ${index} of ${count}`)
  return options
}

export const getRunName = (target, [index, count]) =>
  count > 1 ? `${target.name}-${index}-of-${count}` : target.name

const getWorkers = () => Number(process.env.NUMEN_MUTATION_WORKERS) || availableParallelism()

const run = (command, args, options) => {
  console.log(`$ ${command} ${args.join(' ')}`)
  const result = spawnSync(command, args, { stdio: 'inherit', ...options })
  if (result.error) throw result.error
  return result.status ?? 1
}

/* ---------------------------------- Go ------------------------------------ */

/** The module's packages holding Go source, as `./path` patterns. */
const listGoPackages = (target) => {
  const at = join(root, target.at)
  const listed = execFileSync('go', ['list', '-f', '{{.Dir}} {{join .GoFiles " "}}', './...'], {
    cwd: at,
    encoding: 'utf8',
  })
  return listed
    .split('\n')
    .map((line) => line.split(' '))
    .filter(([dir, ...files]) => dir && files.some((file) => isMutatedGo(file)))
    .map(([dir]) => (dir === at ? '.' : `./${relative(at, dir)}`))
    .sort()
}

const runGo = (target, shard, report, cache) => {
  const packages = pickShard(listGoPackages(target), ...shard)
  if (packages.length === 0) return null
  const args = [
    'run',
    GOMUTANTS,
    '-w',
    String(getWorkers()),
    '-q',
    `--cache=${cache}.gomutants.json`,
  ]
  args.push('--detect-equivalent', '-o', `${cache}.result.json`, '-stryker-output', report)
  args.push('-html-output', report.replace(/\.json$/, '.html'))
  return run('go', [...args, ...packages], { cwd: join(root, target.at) })
}

/* ------------------------------- TypeScript ------------------------------- */

const listSources = (target) =>
  readdirSync(join(root, target.at, 'src'), { recursive: true })
    .map((file) => join(target.at, 'src', file))
    .filter(isMutatedSource)
    .sort()

/**
 * A Vitest config for the target's tests that do not run in a browser, written
 * where Stryker's sandbox keeps it and rooted at the package.
 */
export const renderVitestConfig = (at) => {
  const up = `../../${at.replace(/^modules\//, '')}`
  return [
    "import { fileURLToPath, URL } from 'node:url'",
    "import { defineConfig } from 'vitest/config'",
    `import base from '${up}/vitest.config.ts'`,
    '',
    'const isBrowser = (project) => Boolean(project?.test?.browser?.enabled)',
    'const projects = base.test?.projects',
    '',
    'export default defineConfig({',
    '  ...base,',
    `  root: fileURLToPath(new URL('${up}/', import.meta.url)),`,
    '  test: {',
    '    ...base.test,',
    "    ...(projects ? { projects: projects.filter((one) => typeof one === 'object' && !isBrowser(one)) } : {}),",
    '  },',
    '})',
    '',
  ].join('\n')
}

const writeVitestConfig = (target) => {
  const path = join(root, OUTPUT, 'vitest', `${target.name}.config.ts`)
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, renderVitestConfig(target.at))
  return relative(MODULES, path)
}

const runTs = (target, shard, report, cache) => {
  const mutate = pickShard(listSources(target), ...shard).map((path) => relative('modules', path))
  if (mutate.length === 0) return null
  patchVitestRunner()
  const settings = {
    target,
    vitestConfig: writeVitestConfig(target),
    mutate,
    concurrency: getWorkers(),
    report: relative(MODULES, report),
    html: relative(MODULES, report.replace(/\.json$/, '.html')),
    incrementalFile: relative(MODULES, `${cache}.stryker.json`),
  }
  return run('npx', ['stryker', 'run', join(MODULES, 'tools/mutation/stryker.config.mjs')], {
    cwd: MODULES,
    env: { ...process.env, NUMEN_MUTATION: JSON.stringify(settings) },
  })
}

/* ---------------------------------- Main ---------------------------------- */

const main = () => {
  const { target, shard } = parseArgs(process.argv.slice(2))
  const name = getRunName(target, shard)
  mkdirSync(join(root, OUTPUT, 'reports'), { recursive: true })
  mkdirSync(join(root, OUTPUT, 'cache'), { recursive: true })
  const report = join(root, OUTPUT, 'reports', `${name}.json`)
  const cache = join(root, OUTPUT, 'cache', name)
  rmSync(report, { force: true })
  const status =
    target.kind === 'go' ? runGo(target, shard, report, cache) : runTs(target, shard, report, cache)
  if (status === null) {
    console.log(`${name}: nothing to mutate`)
    writeFileSync(report, JSON.stringify({ schemaVersion: '2', thresholds: {}, files: {} }))
    return
  }
  // No threshold is passed to either tool, so a non-zero exit is a run that could not answer.
  if (status !== 0) {
    console.error(`${name}: the run failed with status ${status}`)
    process.exit(1)
  }
}

if (process.argv[1] === new URL(import.meta.url).pathname) main()
