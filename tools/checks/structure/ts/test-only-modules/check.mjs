import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

/**
 * Runs knip over unused files and resolves to the files it lists, relative to repoRoot.
 *
 * @param {string} repoRoot
 * @param {boolean} production
 * @returns {Promise<{ files: string[], error?: string }>}
 */
function listUnusedFiles(repoRoot, production) {
  const knipBin = path.join(repoRoot, 'node_modules/.bin/knip')
  if (!fs.existsSync(knipBin)) {
    return Promise.resolve({ files: [], error: 'node_modules/.bin/knip is missing; run npm ci' })
  }
  const args = [
    '--no-progress',
    '--reporter',
    'json',
    '--include',
    'files',
    ...(production ? ['--production'] : []),
  ]
  return new Promise((resolve) => {
    const child = spawn(knipBin, args, {
      cwd: repoRoot,
      stdio: ['ignore', 'pipe', 'pipe'],
      maxBuffer: 1 << 28,
    })
    let out = ''
    let err = ''
    child.stdout.on('data', (d) => (out += d.toString()))
    child.stderr.on('data', (d) => (err += d.toString()))
    child.on('error', (cause) => {
      resolve({ files: [], error: `knip did not start: ${cause.message}` })
    })
    child.on('close', (code) => {
      try {
        const report = JSON.parse(out)
        const files = (report.issues ?? [])
          .filter((i) => i.files?.length > 0 || i.files === true)
          .map((i) => i.file)
        resolve({ files })
      } catch {
        resolve({
          files: [],
          error: `knip exited with code ${code} without a report: ${err.trim() || out.trim()}`,
        })
      }
    })
  })
}

/**
 * Lists the glob of every entry of knip.json that is not marked with `!`: these are the entries that
 * only the full run reads (tests, stories, tool configuration), relative to the repository root.
 *
 * @param {string} repoRoot
 * @returns {string[]}
 */
export function listNonProductionEntries(repoRoot) {
  const config = JSON.parse(fs.readFileSync(path.join(repoRoot, 'knip.json'), 'utf8'))
  const globs = []
  const add = (dir, entries) => {
    for (const entry of entries ?? []) {
      if (!entry.endsWith('!') && !entry.startsWith('!')) globs.push(path.posix.join(dir, entry))
    }
  }
  add('.', config.entry)
  for (const [dir, workspace] of Object.entries(config.workspaces ?? {})) add(dir, workspace.entry)
  return globs
}

/**
 * Fails on every file that no production entry reaches although the full run (tests and stories as
 * entries) does: a module only tests import. The files the full run reads as entries (tests,
 * stories and their configuration) and the test support (`testing`, `fixtures` and `testdata`) are not reported, and a file that nothing imports is the finding
 * of the knip check.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runTestOnlyModulesCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now()
  const [shipped, everything] = await Promise.all([
    listUnusedFiles(repoRoot, true),
    listUnusedFiles(repoRoot, false),
  ])
  const error = shipped.error ?? everything.error
  const entries = error ? [] : listNonProductionEntries(repoRoot)
  const unusedAnywhere = new Set(everything.files)
  const found = shipped.files
    .filter((file) => !unusedAnywhere.has(file))
    .filter((file) => !entries.some((glob) => path.matchesGlob(file, glob)))
    .sort()
  const issues = error
    ? [`[TS test-only-modules] ${error}`]
    : found.map((file) => `[TS test-only-modules] ${file}: only tests import this module`)
  return {
    id: 'test-only-modules',
    name: 'Modules Imported Only by Tests (knip --production)',
    pass: issues.length === 0 || warnOnly,
    summary:
      issues.length === 0 ? 'Every module has a shipped importer' : `${issues.length} finding(s)`,
    details: issues.length > 0 ? issues.join('\n') : undefined,
    durationMs: Date.now() - start,
  }
}
