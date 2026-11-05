import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'

const root = resolve(fileURLToPath(import.meta.url), '../../..')
const read = (path) => readFileSync(resolve(root, path), 'utf8')
const mentions = (path, pattern) => pattern.test(read(path))

const rules = [
  '.agents/rules/process.md',
  '.agents/skills/nm-review/stages/2-bughunter.md',
  '.agents/skills/nm-implement/SKILL.md',
]

test('the forward-only script of the index migrations is gone', () => {
  assert.equal(
    existsSync(resolve(root, 'libs/core/adapter/index/migration-forward-only.sh')),
    false,
  )
})

test('no workflow runs a migrations job or the forward-only script', () => {
  const workflow = '.github/workflows/go.yml'
  assert.equal(mentions(workflow, /forward-only/i), false, 'go.yml names forward-only')
  assert.equal(mentions(workflow, /^ {2}migrations:/m), false, 'go.yml has the migrations job')
  assert.equal(mentions(workflow, /name: Migrations/), false, 'go.yml names the Migrations job')
})

test('the Makefile does not run the forward-only script', () => {
  assert.equal(mentions('Makefile', /migration-forward-only/), false)
})

test('the pre-commit hook does not run the forward-only script', () => {
  assert.equal(mentions('tools/git-hooks/.husky/pre-commit', /migration-forward-only/), false)
})

test('the agent rules do not ask for a numbered migration when the schema changes', () => {
  for (const file of rules) {
    assert.equal(
      mentions(file, /numbered migration|migrations are frozen|and migrations/i),
      false,
      file,
    )
  }
})

test('the agent rules say the schema is one file and a change of it changes the version', () => {
  const all = rules.map(read).join('\n')
  assert.match(all, /schema is one file/i)
  assert.match(all, /changes the version/i)
})
