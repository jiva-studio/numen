import assert from 'node:assert/strict'
import { test } from 'node:test'
import { checkVersion, getCodeFiles, getNextVersion, parseVersion } from './check.mjs'

const today = new Date('2026-10-30T12:00:00Z')

test('parseVersion accepts <year>.<month>.<build> only', () => {
  assert.deepEqual(parseVersion('2026.10.3'), { year: 2026, month: 10, build: 3 })
  for (const text of ['1.0.0', '2026.10.0', '2026.13.1', '2026.01.1', '2026.10', '', undefined]) {
    assert.equal(parseVersion(text), null, String(text))
  }
})

test('getNextVersion counts within the month and restarts in a new one', () => {
  assert.equal(getNextVersion('2026.10.4', today), '2026.10.5')
  assert.equal(getNextVersion('2026.9.7', today), '2026.10.1')
  assert.equal(getNextVersion('2025.10.2', today), '2026.10.1')
  assert.equal(getNextVersion('1.0.0', today), '2026.10.1')
  assert.equal(getNextVersion('', today), '2026.10.1')
})

test('getCodeFiles drops markdown and tests', () => {
  assert.deepEqual(
    getCodeFiles(['plugins/a/README.md', 'plugins/a/x.test.ts', 'plugins/a/y_test.go', 'plugins/a/y.go']),
    ['plugins/a/y.go'],
  )
})

test('checkVersion requires a raise only when the plugin changed', () => {
  const base = '2026.10.2'
  assert.equal(checkVersion({ current: base, base, isChanged: false, today }), null)
  assert.match(checkVersion({ current: base, base, isChanged: true, today }), /not raised \(next: 2026.10.3\)/)
  assert.equal(checkVersion({ current: '2026.10.3', base, isChanged: true, today }), null)
})

test('checkVersion accepts a legacy base and a new plugin', () => {
  assert.equal(checkVersion({ current: '2026.10.1', base: '1.0.0', isChanged: true, today }), null)
  assert.equal(checkVersion({ current: '2026.10.1', base: null, isChanged: true, today }), null)
})

test('checkVersion rejects a malformed or future version', () => {
  assert.match(checkVersion({ current: '1.0.0', base: null, isChanged: true, today }), /not <year>/)
  assert.match(checkVersion({ current: '2026.11.1', base: null, isChanged: true, today }), /later than/)
})
