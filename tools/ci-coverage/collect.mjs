#!/usr/bin/env node
/**
 * Measures the statement coverage of every Go module and TypeScript package that
 * `tools/ci-mutation/targets.mjs` finds, and writes the overall percentage, weighted
 * by statements, as coverage.json of the badges gist.
 *
 *   node collect.mjs [--quality <file>]
 *
 * A Go module is measured with `go test -coverprofile ./...`. A TypeScript package or
 * plugin frontend is measured with `vitest run --coverage` against its projects that
 * do not run in a browser, through the config `tools/ci-mutation/mutate.mjs` writes
 * for Stryker. A module whose tests fail counts no statements.
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

/** The modules coverage measures: every mutation target, as `{ name, kind, at }`. */
export const listCoverageModules = () => getTargets().map(({ name, kind, at }) => ({ name, kind, at }))

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

export const runCollection = async (qualityFile = null) => {
  rmSync(tmpDir, { recursive: true, force: true })
  mkdirSync(tmpDir, { recursive: true })

  const measured = listCoverageModules().map(measureModule)
  rmSync(tmpDir, { recursive: true, force: true })

  const badge = getCoverageBadge(measured)
  console.log(`[coverage] Final Unified Coverage: ${badge.message} (${formatPercent(sumStatements(measured))})`)
  for (const kind of ['go', 'ts']) {
    const stats = sumStatements(measured.filter((one) => one.kind === kind))
    console.log(`[coverage]   - ${kind}: ${formatPercent(stats)}`)
  }

  if (qualityFile) writePluginQuality(qualityFile, measured)
  await updateBadgeBranch('coverage.json', badge)
  return { badge, measured }
}

if (process.argv[1] && process.argv[1].endsWith('collect.mjs')) {
  const args = process.argv.slice(2)
  const qualityFile = args.includes('--quality') ? args[args.indexOf('--quality') + 1] : null
  runCollection(qualityFile).catch((err) => {
    console.error(err)
    process.exit(1)
  })
}
