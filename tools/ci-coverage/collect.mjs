#!/usr/bin/env node
/**
 * Collects statement coverage across all Go modules and TypeScript packages,
 * calculates the unified weighted coverage percentage, and updates coverage.json
 * on branch `badges`.
 *
 *   node collect.mjs [--quality <file>]
 *
 * `--quality` also writes the coverage of each plugin module as a quality file of
 * `tools/shared/plugin-quality.mjs`.
 */
import { execTool } from '../shared/tools.mjs'
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { updateBadgeBranch } from '../shared/badge.mjs'
import {
  formatQualityFile,
  isPluginTarget,
  mergeQualityFile,
  roundScore,
} from '../shared/plugin-quality.mjs'
import { getTargets } from '../ci-mutation/targets.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '../..')
const tmpDir = join(root, '.coverage_tmp')

const GO_MODULES = [
  'libs/core/kernel',
  'libs/shared/protocol',
  'libs/shared/identity',
  'libs/shared/syncprotocol',
  'libs/shared/licensing',
  'libs/shared/onnx',
  'libs/shared/tokenizer',
  'apps/desktop',
  'apps/mobile',
  'plugins/agents-numen',
  'plugins/agents-acp',
  'plugins/agents-antigravity',
  'plugins/agents-claude',
  'plugins/agents-codex',
  'plugins/connections-llm',
  'plugins/tools-index-stats',
  'plugins/tools-media-importer',
  'plugins/engines-embeddings',
  'plugins/engines-reranking',
  'plugins/engines-ocr',
  'plugins/tools-proofreading',
  'plugins/tools-saved-search',
  'plugins/tools-search-relevance',
  'plugins/tools-taxonomy',
  'plugins/tools-templates',
  'plugins/tools-sync',
  'plugins/tools-updater',
  'plugins/readers-epub',
  'plugins/readers-pdf',
  'plugins/engines-transcription',
  'plugins/todos',
]

const TS_PACKAGES = ['libs/ui/components', 'apps/desktop/editor', 'apps/desktop/flashcards']

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

/** The mutation target name of a Go module, which names its quality entry. */
const getGoTargetName = (mod) =>
  getTargets().find((one) => one.kind === 'go' && one.at === mod)?.name ?? mod.replaceAll('/', '-')

const collectGo = () => {
  let totalGo = 0
  let coveredGo = 0
  const modules = []

  for (const mod of GO_MODULES) {
    const modDir = join(root, mod)
    if (!existsSync(join(modDir, 'go.mod'))) continue

    const outPath = join(tmpDir, `${mod.replaceAll('/', '_')}.out`)
    try {
      console.log(`[coverage] Testing Go module: ${mod}...`)
      execTool('go', ['test', `-coverprofile=${outPath}`, './...'], {
        cwd: modDir,
        encoding: 'utf8',
        stdio: 'pipe',
      })
      const stats = parseGoProfile(outPath)
      totalGo += stats.total
      coveredGo += stats.covered
      modules.push({ mod, ...stats })
      const pct = stats.total > 0 ? ((100 * stats.covered) / stats.total).toFixed(1) : '0'
      console.log(`[coverage] ${mod}: ${pct}% (${stats.covered}/${stats.total} stmts)`)
    } catch (err) {
      console.warn(`[coverage] Failed running tests in ${mod}:`, err.message)
    }
  }

  return { total: totalGo, covered: coveredGo, modules }
}

/** The coverage entry of each plugin module that has statements. */
export const listPluginCoverage = (modules, getName, measuredAt) =>
  modules
    .filter((one) => one.total > 0)
    .map((one) => ({ name: getName(one.mod), ...one }))
    .filter((one) => isPluginTarget(one.name))
    .map((one) => ({
      target: one.name,
      coverage: roundScore((100 * one.covered) / one.total),
      measuredAt,
    }))

const writePluginQuality = (file, modules) => {
  const measuredAt = new Date().toISOString()
  const entries = listPluginCoverage(modules, getGoTargetName, measuredAt)
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(file, formatQualityFile(mergeQualityFile(null, entries, measuredAt)))
  console.log(`[coverage] ${entries.length} plugin coverage entries written to ${file}`)
}

const collectTs = () => {
  let totalTs = 0
  let coveredTs = 0

  for (const pkg of TS_PACKAGES) {
    const pkgDir = join(root, pkg)
    if (!existsSync(join(pkgDir, 'package.json'))) continue

    try {
      console.log(`[coverage] Testing TS package: ${pkg}...`)
      execTool(
        'npx',
        ['vitest', 'run', '--project', 'unit', '--coverage', '--coverage.reporter=json-summary'],
        {
          cwd: pkgDir,
          encoding: 'utf8',
          stdio: 'pipe',
        },
      )

      const summaryPath = join(pkgDir, 'coverage/coverage-summary.json')
      if (existsSync(summaryPath)) {
        const summary = JSON.parse(readFileSync(summaryPath, 'utf8'))
        const stmts = summary.total?.statements
        if (stmts) {
          totalTs += stmts.total
          coveredTs += stmts.covered
          console.log(`[coverage] ${pkg}: ${stmts.pct}% (${stmts.covered}/${stmts.total} stmts)`)
        }
      }
    } catch (err) {
      console.warn(`[coverage] Failed running tests in ${pkg}:`, err.message)
    }
  }

  return { total: totalTs, covered: coveredTs }
}

export const runCollection = async (qualityFile = null) => {
  rmSync(tmpDir, { recursive: true, force: true })
  mkdirSync(tmpDir, { recursive: true })

  const go = collectGo()
  const ts = collectTs()

  const total = go.total + ts.total
  const covered = go.covered + ts.covered
  const pct = total > 0 ? Math.round((100 * covered) / total) : 0

  console.log(`[coverage] Final Unified Coverage: ${pct}% (Total: ${covered}/${total} statements)`)
  console.log(
    `[coverage]   - Go: ${go.total > 0 ? ((100 * go.covered) / go.total).toFixed(1) : 0}% (${go.covered}/${go.total})`,
  )
  console.log(
    `[coverage]   - TS: ${ts.total > 0 ? ((100 * ts.covered) / ts.total).toFixed(1) : 0}% (${ts.covered}/${ts.total})`,
  )

  rmSync(tmpDir, { recursive: true, force: true })
  if (qualityFile) writePluginQuality(qualityFile, go.modules)

  const color = pct >= 85 ? 'brightgreen' : pct >= 70 ? 'yellow' : 'red'
  const payload = {
    schemaVersion: 1,
    label: 'coverage',
    message: `${pct}%`,
    color,
  }

  await updateBadgeBranch('coverage.json', payload)
  return { pct, go, ts }
}

if (process.argv[1] && process.argv[1].endsWith('collect.mjs')) {
  const args = process.argv.slice(2)
  const qualityFile = args.includes('--quality') ? args[args.indexOf('--quality') + 1] : null
  runCollection(qualityFile).catch((err) => {
    console.error(err)
    process.exit(1)
  })
}
