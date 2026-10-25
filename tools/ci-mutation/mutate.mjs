#!/usr/bin/env node
/**
 * Runs mutation testing on one shard of a target and writes its report in the
 * Stryker schema to `.mutation/reports/`.
 *
 *   node mutate.mjs <target> [--shard 2/8]
 *   node mutate.mjs --file <path>
 *
 * The target's cache in `.mutation/cache/`, which every shard of the
 * target reads and writes, supplies the results whose code and covering tests
 * are unchanged.
 *
 * `--file` mutates one source file of whichever target holds it, with no cache,
 * and prints the mutants its tests do not notice.
 */
import { execFileSync, spawnSync } from 'node:child_process'
import { existsSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs'
import { availableParallelism } from 'node:os'
import { dirname, join, relative, resolve } from 'node:path'
import { getCacheFile, keepFiles } from './cache.mjs'
import { patchVitestRunner } from './patch.mjs'
import { readMutants } from './report.mjs'
import {
  GOMUTANTS,
  OUTPUT,
  getTarget,
  getTargets,
  isMutatedGo,
  isMutatedSource,
  pickShard,
  root,
} from './targets.mjs'

const MODULES = join(root, 'modules')

/** The target holding a repository path: the deepest one of the file's kind. */
export const findTarget = (targets, path) => {
  const kind = path.endsWith('.go') ? 'go' : 'ts'
  const holding = targets
    .filter((one) => one.kind === kind && path.startsWith(`${one.at}/`))
    .sort((a, b) => b.at.length - a.at.length)[0]
  if (!holding) throw new Error(`no mutation target holds ${path}`)
  return holding
}

const parseArgs = (argv) => {
  if (argv[0] === '--file') {
    const file = relative(root, resolve(argv[1] ?? ''))
    if (!isMutatedGo(file) && !isMutatedSource(file)) {
      throw new Error(`${file} is not source a target mutates`)
    }
    return { target: findTarget(getTargets(), file), shard: [1, 1], file }
  }
  const [name, ...rest] = argv
  if (!name) throw new Error('usage: mutate.mjs <target> [--shard k/n] | --file <path>')
  const options = { target: getTarget(name), shard: [1, 1], file: null }
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
    .filter(([dir, ...files]) => dir && files.some((file) => isMutatedGo(join(relative(at, dir), file))))
    .map(([dir]) => (dir === at ? '.' : `./${relative(at, dir)}`))
    .sort()
}

/** The package of one Go file, and patterns for the other files of that package. */
const getGoFile = (target, file) => {
  const own = relative(target.at, file)
  const dir = dirname(own)
  const others = readdirSync(join(root, target.at, dir))
    .map((name) => join(dir, name))
    .filter((path) => isMutatedGo(path) && path !== own)
    .map((path) => `^${path.replaceAll('.', '\\.')}$`)
  return { packages: [dir === '.' ? '.' : `./${dir}`], others }
}

const runGo = (target, shard, { report, cache, result }, file) => {
  const { packages, others } = file
    ? getGoFile(target, file)
    : { packages: pickShard(listGoPackages(target), ...shard), others: [] }
  if (packages.length === 0) return null
  // A mutant is cut off at twice its package's baseline test time.
  const args = ['run', GOMUTANTS, '-w', String(getWorkers()), '-q', '--timeout-coefficient', '2']
  args.push(file ? '--cache=off' : `--cache=${cache}`)
  if (others.length > 0) args.push('-exclude-files', others.join(','))
  args.push('--detect-equivalent', '-o', result, '-stryker-output', report)
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

/** An ES module whatever the package declares, so the configs it imports load as they do in the package. */
const writeVitestConfig = (target) => {
  const path = join(root, OUTPUT, 'vitest', `${target.name}.config.mts`)
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, renderVitestConfig(target.at))
  return relative(MODULES, path)
}

const runTs = (target, shard, { report, cache }, file) => {
  const sources = file ? [file] : pickShard(listSources(target), ...shard)
  const mutate = sources.map((path) => relative('modules', path))
  if (mutate.length === 0) return null
  patchVitestRunner()
  const settings = {
    target,
    vitestConfig: writeVitestConfig(target),
    mutate,
    concurrency: getWorkers(),
    report: relative(MODULES, report),
    html: relative(MODULES, report.replace(/\.json$/, '.html')),
    incrementalFile: file ? null : relative(MODULES, cache),
  }
  const status = run(
    'npx',
    ['stryker', 'run', join(MODULES, 'tools/mutation/stryker.config.mjs')],
    {
      cwd: MODULES,
      env: { ...process.env, NUMEN_MUTATION: JSON.stringify(settings) },
    },
  )
  // Stryker reports every file its incremental file holds; the shard reports the files it mutated.
  if (existsSync(report)) {
    writeFileSync(
      report,
      JSON.stringify(keepFiles(JSON.parse(readFileSync(report, 'utf8')), mutate)),
    )
  }
  return status
}

/* ---------------------------------- Main ---------------------------------- */

/** The mutants its tests did not notice, one line each, in file order. */
export const formatSurvivors = (mutants) =>
  mutants
    .filter((one) => one.status === 'Survived' || one.status === 'NoCoverage')
    .sort((a, b) => a.line - b.line)
    .map(
      (one) =>
        `${one.path}:${one.line} ${one.status === 'NoCoverage' ? 'not covered' : 'survived'} ${one.mutator}: ${one.original} → ${one.replacement}`,
    )

const main = () => {
  const { target, shard, file } = parseArgs(process.argv.slice(2))
  const name = file ? `file-${file.replaceAll('/', '-')}` : getRunName(target, shard)
  mkdirSync(join(root, OUTPUT, 'reports'), { recursive: true })
  mkdirSync(join(root, OUTPUT, 'cache'), { recursive: true })
  const report = join(root, OUTPUT, 'reports', `${name}.json`)
  const paths = {
    report,
    cache: join(root, OUTPUT, 'cache', getCacheFile(target)),
    result: join(root, OUTPUT, 'cache', `${name}.result.json`),
  }
  rmSync(report, { force: true })
  const status =
    target.kind === 'go' ? runGo(target, shard, paths, file) : runTs(target, shard, paths, file)
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
  if (file) {
    const lines = formatSurvivors(readMutants(target, [JSON.parse(readFileSync(report, 'utf8'))]))
    console.log(`\n${lines.length} mutants of ${file} not noticed by its tests`)
    for (const line of lines) console.log(line)
  }
}

if (process.argv[1] === new URL(import.meta.url).pathname) main()
