/**
 * The floor a story run is held to: a run that drew nothing, or fewer than the story files
 * declare, is a fault.
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import { getRunFaults } from './run.mjs'

test('a run that drew every declared story has no fault', () => {
  assert.deepEqual(getRunFaults('webkit', 233, 233), [])
})

test('a run that drew fewer stories than declared names the shortfall', () => {
  assert.deepEqual(getRunFaults('webkit', 233, 13), ['webkit drew 13 stories of the 233 declared'])
})

test('a run over directories that declare nothing is a fault', () => {
  assert.equal(getRunFaults('chromium', 0, 0).length, 1)
})
