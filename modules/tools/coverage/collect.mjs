#!/usr/bin/env node
/**
 * Collects statement coverage across all Go modules and TypeScript packages,
 * calculates the unified weighted coverage percentage, and updates coverage.json
 * on branch `badges`.
 */
import { execFileSync } from 'node:child_process'
import { existsSync, mkdirSync, readFileSync, rmSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { updateBadgeBranch } from '../badge.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '../../..')
const tmpDir = join(root, 'modules/.coverage_tmp')

const GO_MODULES = [
  'modules/libs/core',
  'modules/libs/protocol',
  'modules/apps/desktop',
  'modules/apps/mobile',
  'modules/plugins/agents.numen',
  'modules/plugins/agents.acp',
  'modules/plugins/agents.antigravity',
  'modules/plugins/agents.claude',
  'modules/plugins/agents.codex',
  'modules/plugins/tools.downloader',
  'modules/plugins/engines.embeddings',
  'modules/plugins/engines.ocr',
  'modules/plugins/tools.proofreading',
  'modules/plugins/readers.epub',
  'modules/plugins/readers.pdf',
  'modules/plugins/engines.transcription',
]

const TS_PACKAGES = [
  'modules/libs/ui',
  'modules/apps/desktop/editor',
  'modules/apps/desktop/flashcards',
]

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

const collectGo = () => {
  let totalGo = 0
  let coveredGo = 0

  for (const mod of GO_MODULES) {
    const modDir = join(root, mod)
    if (!existsSync(join(modDir, 'go.mod'))) continue

    const outPath = join(tmpDir, `${mod.replaceAll('/', '_')}.out`)
    try {
      console.log(`[coverage] Testing Go module: ${mod}...`)
      execFileSync('go', ['test', `-coverprofile=${outPath}`, './...'], {
        cwd: modDir,
        encoding: 'utf8',
        stdio: 'pipe',
      })
      const stats = parseGoProfile(outPath)
      totalGo += stats.total
      coveredGo += stats.covered
      const pct = stats.total > 0 ? ((100 * stats.covered) / stats.total).toFixed(1) : '0'
      console.log(`[coverage] ${mod}: ${pct}% (${stats.covered}/${stats.total} stmts)`)
    } catch (err) {
      console.warn(`[coverage] Failed running tests in ${mod}:`, err.message)
    }
  }

  return { total: totalGo, covered: coveredGo }
}

const collectTs = () => {
  let totalTs = 0
  let coveredTs = 0

  for (const pkg of TS_PACKAGES) {
    const pkgDir = join(root, pkg)
    if (!existsSync(join(pkgDir, 'package.json'))) continue

    try {
      console.log(`[coverage] Testing TS package: ${pkg}...`)
      execFileSync('npx', ['vitest', 'run', '--project', 'unit', '--coverage', '--coverage.reporter=json-summary'], {
        cwd: pkgDir,
        encoding: 'utf8',
        stdio: 'pipe',
      })

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

export const runCollection = async () => {
  rmSync(tmpDir, { recursive: true, force: true })
  mkdirSync(tmpDir, { recursive: true })

  const go = collectGo()
  const ts = collectTs()

  const total = go.total + ts.total
  const covered = go.covered + ts.covered
  const pct = total > 0 ? Math.round((100 * covered) / total) : 0

  console.log(`[coverage] Final Unified Coverage: ${pct}% (Total: ${covered}/${total} statements)`)
  console.log(`[coverage]   - Go: ${go.total > 0 ? ((100 * go.covered) / go.total).toFixed(1) : 0}% (${go.covered}/${go.total})`)
  console.log(`[coverage]   - TS: ${ts.total > 0 ? ((100 * ts.covered) / ts.total).toFixed(1) : 0}% (${ts.covered}/${ts.total})`)

  rmSync(tmpDir, { recursive: true, force: true })

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
  runCollection().catch((err) => {
    console.error(err)
    process.exit(1)
  })
}
