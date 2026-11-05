/**
 * The palette asks only the required arguments of a command, so a manifest has no flag that keeps an
 * argument out of it: no tracked file names it, generated code included.
 */
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import test from 'node:test'
import { root } from './modules.mjs'

const REMOVED_FLAG = new RegExp(['hidden', 'in', 'palette'].join('_?'), 'i')

const listTrackedFiles = () =>
  execFileSync('git', ['ls-files', '-z'], { cwd: root, maxBuffer: 1 << 28 })
    .toString()
    .split('\0')
    .filter(Boolean)

test('no tracked file names the flag that hid an argument from the palette', () => {
  const naming = listTrackedFiles().filter((file) => {
    try {
      return REMOVED_FLAG.test(readFileSync(join(root, file), 'utf8'))
    } catch {
      return false
    }
  })

  assert.deepEqual(naming, [])
})
