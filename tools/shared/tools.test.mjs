import assert from 'node:assert/strict'
import { chmodSync, mkdtempSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { delimiter, join } from 'node:path'
import { test } from 'node:test'
import { execTool, findTool, isTrustedDirectory, runTool } from './tools.mjs'

const makeProgram = (dir, name) => {
  const file = join(dir, name)
  writeFileSync(file, '#!/bin/sh\necho ok\n')
  chmodSync(file, 0o755)
  return file
}

test('a relative or temporary PATH entry is not trusted', () => {
  assert.equal(isTrustedDirectory('.', {}), false)
  assert.equal(isTrustedDirectory('bin', {}), false)
  assert.equal(isTrustedDirectory(tmpdir(), {}), false)
  assert.equal(isTrustedDirectory('/usr/bin', {}), true)
  assert.equal(isTrustedDirectory('/nix/store/abc-git/bin', {}), true)
})

test('a program in an untrusted directory is not found', () => {
  const dir = mkdtempSync(join(tmpdir(), 'tool-untrusted-'))
  makeProgram(dir, 'numen-fake-tool')
  assert.equal(findTool('numen-fake-tool', { PATH: dir }), null)
})

test('NUMEN_TOOL_PATH trusts a directory', () => {
  const dir = mkdtempSync(join(tmpdir(), 'tool-trusted-'))
  const file = makeProgram(dir, 'numen-fake-tool')
  const env = { PATH: dir, NUMEN_TOOL_PATH: ['/elsewhere', dir].join(delimiter) }
  assert.equal(findTool('numen-fake-tool', env), file)
})

test('runTool reports a missing program in error and execTool throws', () => {
  const saved = process.env.PATH
  process.env.PATH = tmpdir()
  try {
    assert.match(runTool('numen-no-such-tool', []).error.message, /trusted directory/)
    assert.throws(() => execTool('numen-no-such-tool', []), /trusted directory/)
  } finally {
    process.env.PATH = saved
  }
})

test('runTool runs a program found on the real PATH', () => {
  const run = runTool('git', ['--version'], { encoding: 'utf8' })
  assert.equal(run.status, 0)
  assert.match(run.stdout, /^git version/)
})
