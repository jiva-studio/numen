/**
 * The cell of a table row in the reference of the manual.
 *
 * Run with `node --test apps/docs/tools/cell.test.mjs` from the root of the repository.
 */
import assert from 'node:assert/strict'
import test from 'node:test'

import { cell } from './cell.mjs'

test('escapes every bar of the text', () => {
  assert.equal(cell('a | b | c'), 'a \\| b \\| c')
})

test('escapes a backslash so that it cannot escape the bar after it', () => {
  assert.equal(cell('a \\| b'), 'a \\\\\\| b')
})

test('escapes every backslash of the text', () => {
  assert.equal(cell('C:\\one\\two'), 'C:\\\\one\\\\two')
})

test('joins the lines of the text with one space', () => {
  assert.equal(cell('one\n  two\n\nthree'), 'one two three')
})
