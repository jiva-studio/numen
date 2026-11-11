#!/usr/bin/env node
/**
 * Conventional Commits & Scope Validator for Numen (@numen/git.hooks)
 *
 * Ensures a pull request title follows Conventional Commits with a type from ALLOWED_TYPES and, when
 * present, one or more comma-separated scopes of lowercase words joined by hyphens:
 *   <type>(<scope>): <subject>
 *   <type>: <subject>
 *
 * The title is the first argument or, without one, the environment variable PR_TITLE.
 * Commits on a branch are not validated.
 */
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

/** A scope is lowercase words joined by hyphens: `core`, `plugin-sdk`, `reader-epub`. */
export const SCOPE_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/

export function validateCommitMessage(content) {
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

  const match = header.match(/^([a-z]+)(?:\(([^)]+)\))?(!)?:\s(.+)$/)

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
      if (!SCOPE_PATTERN.test(scope)) {
        return {
          ok: false,
          error: `Malformed scope: "(${scope})". A scope is lowercase words joined by hyphens.`,
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

/** Validates the pull request title given as the first argument or in PR_TITLE. */
function main() {
  const content = process.argv[2] ?? process.env.PR_TITLE ?? ''
  const result = validateCommitMessage(content)
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
