import test from 'node:test'
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { ALLOWED_TYPES, validateCommitMessage } from './check-commit.mjs'

test('ALLOWED_TYPES contains standard conventional commit types', () => {
  assert.ok(ALLOWED_TYPES.has('feat'))
  assert.ok(ALLOWED_TYPES.has('fix'))
  assert.ok(ALLOWED_TYPES.has('refactor'))
  assert.ok(ALLOWED_TYPES.has('test'))
  assert.ok(ALLOWED_TYPES.has('arch'))
  assert.ok(ALLOWED_TYPES.has('sec'))
})

test('validateCommitMessage accepts valid conventional commit messages', () => {
  const validMessages = [
    'feat(ocr): add support for orientation detection',
    'fix(transcription): handle empty audio stream',
    'refactor(flashcards): replace note buffer with cards store',
    'test(agent-claude): eliminate mutation survivors',
    'fix(reader-epub): parse toc correctly',
    'arch(desktop): register plugin host',
    'docs: update installation instructions',
    'chore(deps): bump vite to 5.0.5',
    'feat(core,desktop): coordinate active tab state',
    'refactor(core)!: drop legacy sync endpoint',
  ]

  for (const msg of validMessages) {
    const res = validateCommitMessage(msg)
    assert.equal(res.ok, true, `Expected valid for: "${msg}", got error: ${res.error}`)
  }
})

test('validateCommitMessage accepts merge and revert commits', () => {
  assert.equal(
    validateCommitMessage("Merge branch 'main' into feature").ok,
    true
  )
  assert.equal(
    validateCommitMessage('Revert "feat(ui): add modal"').ok,
    true
  )
})

test('validateCommitMessage rejects invalid types and unknown scopes', () => {
  const invalidType = validateCommitMessage(
    'badtype(core): do something'
  )
  assert.equal(invalidType.ok, false)
  assert.match(invalidType.error, /Unknown commit type/)

  const malformedScope = validateCommitMessage('feat(Non_Existent Scope): do something')
  assert.equal(malformedScope.ok, false)
  assert.match(malformedScope.error, /Malformed scope/)
  assert.equal(validateCommitMessage('feat(any-new-scope): do something').ok, true)
  assert.equal(validateCommitMessage('feat(core,): do something').ok, false)

  const emptySubject = validateCommitMessage('feat(core):')
  assert.equal(emptySubject.ok, false)

  assert.equal(validateCommitMessage('feat(core):   wide gap').ok, true)
  assert.equal(validateCommitMessage('feat(core):subject').ok, false)

  const emptyHeader = validateCommitMessage('')
  assert.equal(emptyHeader.ok, false)

  const tooLong = validateCommitMessage(
    `feat(core): ${'a'.repeat(95)}`
  )
  assert.equal(tooLong.ok, false)
  assert.match(tooLong.error, /too long/)
})

test('the command validates the title given as an argument or in PR_TITLE', () => {
  const script = path.join(path.dirname(fileURLToPath(import.meta.url)), 'check-commit.mjs')
  const run = (args, env) =>
    spawnSync('node', [script, ...args], { encoding: 'utf8', env: { ...process.env, PR_TITLE: '', ...env } })

  assert.equal(run(['feat(core): add a port'], {}).status, 0)
  assert.equal(run(['badtype(core): add a port'], {}).status, 1)
  assert.equal(run([], { PR_TITLE: 'fix(tools): read a title' }).status, 0)
  assert.equal(run([], { PR_TITLE: 'bad title' }).status, 1)
  assert.equal(run([], {}).status, 1)
})
