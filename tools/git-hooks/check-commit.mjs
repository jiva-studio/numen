#!/usr/bin/env node
/**
 * Conventional Commits & Scope Validator for Numen (@numen/git.hooks)
 *
 * Ensures all commit messages strictly follow Conventional Commits:
 *   <type>(<scope>): <subject>
 *   <type>: <subject>
 */
import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

export const ALLOWED_TYPES = new Set([
  'feat',
  'fix',
  'docs',
  'refactor',
  'perf',
  'test',
  'build',
  'ci',
  'chore',
  'revert',
  'sec',
  'arch',
])

export const ALLOWED_BASE_SCOPES = new Set([
  // Core packages & layers
  'core',
  'domain',
  'notes',
  'ui',
  'wire',
  'protocol',
  'sdk',
  'plugin-sdk',
  'plugin-sdk-go',
  'plugin-sdk-ts',

  // Apps
  'desktop',
  'editor',
  'flashcards',
  'mobile',
  'landing',
  'docs',
  'cli',

  // Infrastructure & quality
  'tools',
  'lint',
  'ci',
  'deps',
  'repo',
  'release',
  'arch',
  'e2e',
  'test',
  'tests',
  'quality',
  'mutation',
  'workflows',
  'build',

  // Concepts & domains
  'vault',
  'mcp',
  'frontend',
  'backend',
  'security',

  // Categories (general fallback)
  'plugins',
  'plugin',
  'agent',
  'agents',
  'engine',
  'engines',
  'reader',
  'readers',
  'view',
  'views',
  'editors',
])

export function getDynamicScopes(repoRoot) {
  const dynamic = new Set(ALLOWED_BASE_SCOPES)
  const scanDirs = [
    { dir: 'plugins', isPlugin: true },
    { dir: 'apps' },
    { dir: 'libs' },
    { dir: 'tools' },
  ]

  for (const { dir, isPlugin } of scanDirs) {
    const fullDir = path.join(repoRoot, dir)
    if (!fs.existsSync(fullDir)) continue
    try {
      const entries = fs.readdirSync(fullDir, { withFileTypes: true })
      for (const entry of entries) {
        if (!entry.isDirectory() || entry.name.startsWith('.')) continue
        const name = entry.name
        dynamic.add(name)

        if (isPlugin) {
          const parts = name.split('.')
          const prefix = parts[0]
          const singularPrefix = prefix.endsWith('s') ? prefix.slice(0, -1) : prefix

          dynamic.add(prefix)
          dynamic.add(singularPrefix)

          if (parts.length > 1) {
            const rest = parts.slice(1)
            dynamic.add(rest.join('.'))
            dynamic.add(rest.join('-'))
            dynamic.add(parts.join('-'))
            dynamic.add(`${singularPrefix}-${rest.join('-')}`)

            for (const part of rest) {
              dynamic.add(part)
            }
          }
        }
      }
    } catch {
      // Fallback to base scopes if directory cannot be read
    }
  }

  return dynamic
}

export function validateCommitMessage(content, repoRoot) {
  const lines = content.split('\n')
  const headerLine = lines.find((l) => {
    const t = l.trim()
    return t.length > 0 && !t.startsWith('#')
  })

  if (!headerLine) {
    return { ok: false, error: 'Commit message header is empty.' }
  }

  const header = headerLine.trim()

  if (
    header.startsWith('Merge ') ||
    header.startsWith('Revert "') ||
    header.startsWith('fixup! ') ||
    header.startsWith('squash! ')
  ) {
    return { ok: true }
  }

  const allowedScopes = getDynamicScopes(repoRoot)
  const match = header.match(/^([a-z]+)(?:\(([^)]+)\))?(!)?:\s+(.+)$/)

  if (!match) {
    return {
      ok: false,
      error: `Invalid commit message header: "${header}". Must follow Conventional Commits format: <type>(<scope>): <subject> or <type>: <subject>`,
    }
  }

  const [, type, rawScope, , subject] = match

  if (!ALLOWED_TYPES.has(type)) {
    return {
      ok: false,
      error: `Unknown commit type: "${type}". Allowed types: ${Array.from(ALLOWED_TYPES).join(', ')}`,
    }
  }

  if (rawScope) {
    const scopes = rawScope.split(',').map((s) => s.trim())
    for (const scope of scopes) {
      if (!allowedScopes.has(scope) && !allowedScopes.has(scope.toLowerCase())) {
        return {
          ok: false,
          error: `Unknown scope: "(${scope})".`,
        }
      }
    }
  }

  if (header.length > 100) {
    return {
      ok: false,
      error: `Commit header is too long (${header.length} characters, max 100).`,
    }
  }

  if (!subject || subject.trim().length === 0) {
    return { ok: false, error: 'Commit subject cannot be empty.' }
  }

  return { ok: true }
}

function main() {
  const msgFile = process.argv[2]
  if (!msgFile || !fs.existsSync(msgFile)) {
    console.error('❌ Error: No commit message file provided.')
    process.exit(1)
  }

  const content = fs.readFileSync(msgFile, 'utf8')
  const scriptDir = path.dirname(new URL(import.meta.url).pathname)
  const repoRoot = path.resolve(scriptDir, '../../..')

  const result = validateCommitMessage(content, repoRoot)
  if (!result.ok) {
    console.error(`\n❌ Error: ${result.error}\n`)
    process.exit(1)
  }
}

const isDirectRun =
  process.argv[1] && new URL(import.meta.url).pathname === path.resolve(process.argv[1])
if (isDirectRun) {
  main()
}
