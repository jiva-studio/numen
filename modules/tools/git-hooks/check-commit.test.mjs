import test from 'node:test'
import assert from 'node:assert/strict'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  ALLOWED_TYPES,
  getDynamicScopes,
  validateCommitMessage,
} from './check-commit.mjs'

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../../..'
)

test('ALLOWED_TYPES contains standard conventional commit types', () => {
  assert.ok(ALLOWED_TYPES.has('feat'))
  assert.ok(ALLOWED_TYPES.has('fix'))
  assert.ok(ALLOWED_TYPES.has('refactor'))
  assert.ok(ALLOWED_TYPES.has('test'))
  assert.ok(ALLOWED_TYPES.has('arch'))
  assert.ok(ALLOWED_TYPES.has('sec'))
})

test('getDynamicScopes discovers idiomatic plugin names and variants', () => {
  const scopes = getDynamicScopes(repoRoot)

  // Discovered from engines.*
  assert.ok(scopes.has('ocr'))
  assert.ok(scopes.has('transcription'))
  assert.ok(scopes.has('embeddings'))
  assert.ok(scopes.has('engines.ocr'))
  assert.ok(scopes.has('engines-ocr'))
  assert.ok(scopes.has('engine-ocr'))

  // Discovered from readers.*
  assert.ok(scopes.has('epub'))
  assert.ok(scopes.has('pdf'))
  assert.ok(scopes.has('reader-epub'))
  assert.ok(scopes.has('reader-pdf'))

  // Discovered from agents.*
  assert.ok(scopes.has('claude'))
  assert.ok(scopes.has('codex'))
  assert.ok(scopes.has('antigravity'))
  assert.ok(scopes.has('agent-claude'))
  assert.ok(scopes.has('agent-codex'))

  // Discovered from views.*
  assert.ok(scopes.has('flashcards'))
  assert.ok(scopes.has('flashcards-studio'))
  assert.ok(scopes.has('plex'))
  assert.ok(scopes.has('plex-graph'))

  // Discovered from apps
  assert.ok(scopes.has('desktop-monolith'))
  assert.ok(scopes.has('desktop'))

  // Base scopes
  assert.ok(scopes.has('core'))
  assert.ok(scopes.has('domain'))
  assert.ok(scopes.has('ui'))
})

test('validateCommitMessage accepts valid conventional commit messages', () => {
  const validMessages = [
    'feat(ocr): add support for orientation detection',
    'fix(transcription): handle empty audio stream',
    'refactor(flashcards): replace note buffer with cards store',
    'test(agent-claude): eliminate mutation survivors',
    'fix(reader-epub): parse toc correctly',
    'arch(desktop-monolith): register builtin plugins',
    'docs: update installation instructions',
    'chore(deps): bump vite to 5.0.5',
    'feat(core,desktop): coordinate active tab state',
    'refactor(core)!: drop legacy sync endpoint',
  ]

  for (const msg of validMessages) {
    const res = validateCommitMessage(msg, repoRoot)
    assert.equal(res.ok, true, `Expected valid for: "${msg}", got error: ${res.error}`)
  }
})

test('validateCommitMessage accepts merge and revert commits', () => {
  assert.equal(
    validateCommitMessage("Merge branch 'main' into feature", repoRoot).ok,
    true
  )
  assert.equal(
    validateCommitMessage('Revert "feat(ui): add modal"', repoRoot).ok,
    true
  )
})

test('validateCommitMessage rejects invalid types and unknown scopes', () => {
  const invalidType = validateCommitMessage(
    'badtype(core): do something',
    repoRoot
  )
  assert.equal(invalidType.ok, false)
  assert.match(invalidType.error, /Unknown commit type/)

  const unknownScope = validateCommitMessage(
    'feat(non_existent_scope_12345): do something',
    repoRoot
  )
  assert.equal(unknownScope.ok, false)
  assert.match(unknownScope.error, /Unknown scope/)

  const emptySubject = validateCommitMessage('feat(core):', repoRoot)
  assert.equal(emptySubject.ok, false)

  const emptyHeader = validateCommitMessage('', repoRoot)
  assert.equal(emptyHeader.ok, false)

  const tooLong = validateCommitMessage(
    `feat(core): ${'a'.repeat(95)}`,
    repoRoot
  )
  assert.equal(tooLong.ok, false)
  assert.match(tooLong.error, /too long/)
})
