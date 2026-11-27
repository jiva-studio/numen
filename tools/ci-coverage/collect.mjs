#!/usr/bin/env node
/**
 * Measures the statement coverage of every Go module and TypeScript package that
 * `tools/ci-mutation/targets.mjs` finds, and writes the overall percentage, weighted
 * by statements, as coverage.json of the badges gist.
 *
 *   node collect.mjs [--quality <file>]
 *   node collect.mjs --kind go|ts --out <file>
 *   node collect.mjs --merge <file> <file> [--quality <file>]
 *
 * A Go module is measured with `go test -coverprofile ./...`. A TypeScript package or
 * plugin frontend is measured with `vitest run --coverage` against its projects that
 * do not run in a browser, through the config `tools/ci-mutation/mutate.mjs` writes
 * for Stryker. A module whose tests fail counts no statements.
 *
 * `--kind` measures the modules of one kind and writes them with their statements to
 * `--out`, and leaves the badge as it is. `--merge` reads one such file of each kind
 * and writes the badge and the quality file from all their modules.
 *
 * `--quality` also writes the coverage of each plugin target as a quality file of
 * `tools/shared/plugin-quality.mjs`.
 */
import { execTool } from '../shared/tools.mjs'
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { updateBadgeBranch } from '../shared/badge.mjs'
import {
  formatQualityFile,
  isPluginTarget,
  mergeQualityFile,
  roundScore,
} from '../shared/plugin-quality.mjs'
import { getTargets, root } from '../ci-mutation/targets.mjs'
import { renderVitestConfig } from '../ci-mutation/mutate.mjs'

const TMP = '.coverage_tmp'
const tmpDir = join(root, TMP)

export const KINDS = ['go', 'ts']

/** The modules coverage measures: every mutation target, or those of one kind, as `{ name, kind, at }`. */
export const listCoverageModules = (kind = null) =>
  getTargets()
    .filter((one) => kind === null || one.kind === kind)
    .map(({ name, kind, at }) => ({ name, kind, at }))

/**
 * Parses a Go coverprofile file and returns { total, covered } statements.
 */
export const parseGoProfile = (profilePath) => {
  if (!existsSync(profilePath)) return { total: 0, covered: 0 }
  const lines = readFileSync(profilePath, 'utf8').split('\n')
  let total = 0
  let covered = 0

  for (const line of lines) {
    if (!line || line.startsWith('mode:')) continue
    const parts = line.trim().split(' ')
    if (parts.length === 3) {
      const stmts = Number.parseInt(parts[1], 10)
      const count = Number.parseInt(parts[2], 10)
      if (!Number.isNaN(stmts)) {
        total += stmts
        if (count > 0) covered += stmts
      }
    }
  }

  return { total, covered }
}

const measureGo = (module) => {
  const profile = join(tmpDir, `${module.name}.out`)
  execTool('go', ['test', `-coverprofile=${profile}`, './...'], {
    cwd: join(root, module.at),
    encoding: 'utf8',
    stdio: 'pipe',
  })
  return parseGoProfile(profile)
}

/** `renderVitestConfig` reaches the package from a config two folders below the repository. */
const measureTs = (module) => {
  const config = join(TMP, 'vitest', `${module.name}.config.mts`)
  const reports = join(tmpDir, 'reports', module.name)
  mkdirSync(join(root, TMP, 'vitest'), { recursive: true })
  writeFileSync(join(root, config), renderVitestConfig(module.at))
  execTool(
    'npx',
    [
      'vitest',
      'run',
      '--config',
      config,
      '--coverage',
      '--coverage.reporter=json-summary',
      `--coverage.reportsDirectory=${reports}`,
    ],
    { cwd: root, encoding: 'utf8', stdio: 'pipe' },
  )
  const summary = join(reports, 'coverage-summary.json')
  if (!existsSync(summary)) return { total: 0, covered: 0 }
  const statements = JSON.parse(readFileSync(summary, 'utf8')).total?.statements
  return { total: statements?.total ?? 0, covered: statements?.covered ?? 0 }
}

const formatPercent = ({ total, covered }) =>
  `${total > 0 ? ((100 * covered) / total).toFixed(1) : 0}% (${covered}/${total} stmts)`

const measureModule = (module) => {
  console.log(`[coverage] Testing ${module.kind} module: ${module.at}...`)
  const started = Date.now()
  try {
    const stats = module.kind === 'go' ? measureGo(module) : measureTs(module)
    const seconds = Math.round((Date.now() - started) / 1000)
    console.log(`[coverage] ${module.name}: ${formatPercent(stats)} in ${seconds}s`)
    return { ...module, ...stats }
  } catch (err) {
    console.warn(`[coverage] Failed running tests in ${module.at}:`, err.message)
    return { ...module, total: 0, covered: 0 }
  }
}

const sumStatements = (measured) =>
  measured.reduce(
    (sum, one) => ({ total: sum.total + one.total, covered: sum.covered + one.covered }),
    { total: 0, covered: 0 },
  )

/** The coverage badge of the measured modules, weighted by their statements. */
export const getCoverageBadge = (measured) => {
  const { total, covered } = sumStatements(measured)
  const pct = total > 0 ? Math.round((100 * covered) / total) : 0
  const color = pct >= 85 ? 'brightgreen' : pct >= 70 ? 'yellow' : 'red'
  return { schemaVersion: 1, label: 'coverage', message: `${pct}%`, color }
}

/** The coverage entry of each measured plugin target that has statements. */
export const listPluginCoverage = (measured, measuredAt) =>
  measured
    .filter((one) => isPluginTarget(one.name) && one.total > 0)
    .map((one) => ({
      target: one.name,
      coverage: roundScore((100 * one.covered) / one.total),
      measuredAt,
    }))

const writePluginQuality = (file, measured) => {
  const measuredAt = new Date().toISOString()
  const entries = listPluginCoverage(measured, measuredAt)
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(file, formatQualityFile(mergeQualityFile(null, entries, measuredAt)))
  console.log(`[coverage] ${entries.length} plugin coverage entries written to ${file}`)
}

const measureModules = (kind = null) => {
  rmSync(tmpDir, { recursive: true, force: true })
  mkdirSync(tmpDir, { recursive: true })
  const measured = listCoverageModules(kind).map(measureModule)
  rmSync(tmpDir, { recursive: true, force: true })
  return measured
}

/** The content of the file `--kind` writes: the measured modules of that kind. */
export const formatMeasuredFile = (kind, measured) =>
  `${JSON.stringify({ kind, measured: measured.filter((one) => one.kind === kind) }, null, 2)}\n`

/** The measured modules of the files `--kind` writes; the files hold each kind once. */
export const readMeasuredFiles = (files) => {
  const results = files.map((file) => JSON.parse(readFileSync(file, 'utf8')))
  const kinds = results.map((result) => result.kind).sort()
  if (kinds.join(',') !== [...KINDS].sort().join(',')) {
    throw new Error(`--merge needs one file of each kind (${KINDS.join(', ')}), got: ${kinds.join(', ')}`)
  }
  return results.flatMap((result) => result.measured)
}

const writeMeasuredFile = (file, kind, measured) => {
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(file, formatMeasuredFile(kind, measured))
  console.log(`[coverage] ${kind}: ${formatPercent(sumStatements(measured))} written to ${file}`)
}

const publishCoverage = async (measured, qualityFile) => {
  const badge = getCoverageBadge(measured)
  console.log(`[coverage] Final Unified Coverage: ${badge.message} (${formatPercent(sumStatements(measured))})`)
  for (const kind of KINDS) {
    const stats = sumStatements(measured.filter((one) => one.kind === kind))
    console.log(`[coverage]   - ${kind}: ${formatPercent(stats)}`)
  }

  if (qualityFile) writePluginQuality(qualityFile, measured)
  await updateBadgeBranch('coverage.json', badge)
  return badge
}

/** The command line options as `{ kind, out, merge, quality }`; `merge` lists the files after `--merge`. */
export const readOptions = (args) => {
  const readValue = (name) => (args.includes(name) ? args[args.indexOf(name) + 1] : null)
  const readList = (name) => {
    if (!args.includes(name)) return null
    const rest = args.slice(args.indexOf(name) + 1)
    const end = rest.findIndex((one) => one.startsWith('--'))
    return end === -1 ? rest : rest.slice(0, end)
  }
  const options = {
    kind: readValue('--kind'),
    out: readValue('--out'),
    merge: readList('--merge'),
    quality: readValue('--quality'),
  }
  if (options.kind !== null && !KINDS.includes(options.kind)) {
    throw new Error(`--kind takes ${KINDS.join(' or ')}, got: ${options.kind}`)
  }
  if (options.kind !== null && !options.out) throw new Error('--kind needs --out <file>')
  if (options.kind !== null && options.merge !== null) throw new Error('--kind and --merge exclude each other')
  if (options.kind !== null && options.quality !== null) throw new Error('--quality goes with --merge, not --kind')
  if (options.merge !== null && options.merge.length === 0) throw new Error('--merge needs the files --kind writes')
  return options
}

export const runCollection = async (qualityFile = null) => {
  const measured = measureModules()
  const badge = await publishCoverage(measured, qualityFile)
  return { badge, measured }
}

const runCommand = async ({ kind, out, merge, quality }) => {
  if (kind !== null) {
    writeMeasuredFile(out, kind, measureModules(kind))
  } else if (merge !== null) {
    await publishCoverage(readMeasuredFiles(merge), quality)
  } else {
    await runCollection(quality)
  }
}

if (process.argv[1] && process.argv[1].endsWith('collect.mjs')) {
  Promise.resolve()
    .then(() => runCommand(readOptions(process.argv.slice(2))))
    .catch((err) => {
      console.error(err)
      process.exit(1)
    })
}
