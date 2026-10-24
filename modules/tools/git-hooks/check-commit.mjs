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

const ALLOWED_TYPES = new Set([
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

const ALLOWED_BASE_SCOPES = new Set([
  'core',
  'domain',
  'notes',
  'ui',
  'wire',
  'protocol',
  'desktop',
  'editor',
  'flashcards',
  'mobile',
  'landing',
  'docs',
  'plugins',
  'sdk',
  'plugin-sdk',
  'plugin-sdk-go',
  'plugin-sdk-ts',
  'tools',
  'lint',
  'agent',
  'agents',
  'ci',
  'deps',
  'repo',
  'release',
  'arch',
  'e2e',
  'test',
  'tests',
  'vault',
  'workflows',
  'quality',
  'cli',
])

function getDynamicScopes(repoRoot) {
  const dynamic = new Set(ALLOWED_BASE_SCOPES)
  const pluginsDir = path.join(repoRoot, 'modules/plugins')
  if (fs.existsSync(pluginsDir)) {
    try {
      const entries = fs.readdirSync(pluginsDir, { withFileTypes: true })
      for (const entry of entries) {
        if (entry.isDirectory() && !entry.name.startsWith('.')) {
          dynamic.add(entry.name)
          const prefix = entry.name.split('.')[0]
          if (prefix) dynamic.add(prefix)
        }
      }
    } catch {
      // Fallback to base scopes if plugins cannot be read
    }
  }
  return dynamic
}

function main() {
  const msgFile = process.argv[2]
  if (!msgFile || !fs.existsSync(msgFile)) {
    console.error('❌ Error: No commit message file provided.')
    process.exit(1)
  }

  const content = fs.readFileSync(msgFile, 'utf8')
  const lines = content.split('\n')

  // Find first non-comment, non-empty line
  const headerLine = lines.find((l) => {
    const t = l.trim()
    return t.length > 0 && !t.startsWith('#')
  })

  if (!headerLine) {
    console.error('❌ Error: Commit message header is empty.')
    process.exit(1)
  }

  const header = headerLine.trim()

  // Ignore automatic merge or revert commits
  if (
    header.startsWith('Merge ') ||
    header.startsWith('Revert "') ||
    header.startsWith('fixup! ') ||
    header.startsWith('squash! ')
  ) {
    process.exit(0)
  }

  const scriptDir = path.dirname(new URL(import.meta.url).pathname)
  const repoRoot = path.resolve(scriptDir, '../../..')
  const allowedScopes = getDynamicScopes(repoRoot)

  const match = header.match(/^([a-z]+)(?:\(([^)]+)\))?(!)?:\s+(.+)$/)
  if (!match) {
    console.error(`
❌ Invalid commit message header!
   Header: "${header}"

Commit messages must follow Conventional Commits format:
   <type>(<scope>): <subject>
   <type>: <subject>

Examples:
   feat(ui): add virtual list to command palette
   fix(desktop): preserve active tab on window blur
   refactor(plugins): decompose epub chapter loader
`)
    process.exit(1)
  }

  const [, type, rawScope, breaking, subject] = match

  if (!ALLOWED_TYPES.has(type)) {
    console.error(`
❌ Unknown commit type: "${type}"
   Allowed types: ${Array.from(ALLOWED_TYPES).join(', ')}
`)
    process.exit(1)
  }

  if (rawScope) {
    const scopes = rawScope.split(',').map((s) => s.trim())
    for (const scope of scopes) {
      if (!allowedScopes.has(scope) && !allowedScopes.has(scope.toLowerCase())) {
        console.error(`
❌ Unknown scope: "(${scope})"
   Allowed scopes include:
   - Packages: core, domain, notes, ui, wire, protocol, sdk, desktop, mobile, landing, docs
   - Apps: editor, flashcards, cli
   - Infrastructure: tools, lint, ci, deps, repo, release, arch, test
   - Plugins: plugins, or specific plugin (e.g. readers.epub, views.file.tree)
   - Multi-scope: comma-separated (e.g. ui,desktop)
`)
        process.exit(1)
      }
    }
  }

  if (header.length > 100) {
    console.error(`
❌ Commit header is too long (${header.length} characters).
   Maximum allowed length is 100 characters (ideally <= 72 characters).
`)
    process.exit(1)
  }

  if (!subject || subject.trim().length === 0) {
    console.error('❌ Commit subject cannot be empty.')
    process.exit(1)
  }
}

main()
