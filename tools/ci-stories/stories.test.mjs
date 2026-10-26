/**
 * The rule the two pages hold their shots to, and that the walk behind it
 * reads the tree at all.
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import { REFUSES, faults, proves, stories } from './stories.mjs'

test('the rule refuses what the table says it refuses, and nothing else', () => {
  assert.equal(proves(), null)
})

test('the table walks both paths, so neither is an assumption', () => {
  assert.ok(REFUSES.some((one) => one.said.length > 0))
  assert.ok(REFUSES.some((one) => one.said.length === 0))
})

test('a fault says which story it could not find', () => {
  assert.deepEqual(faults([{ story: 'flash-cards-window--owing' }], new Map([['a--b', 'ui']])), [
    'no story called flash-cards-window--owing — the pictures cannot be taken again',
  ])
})

/**
 * A walk that read the wrong folders finds nothing and says so in the same
 * words as a walk that read the right ones, so the stories it must have found
 * are named — one from the library, one from a window and one from a plugin —
 * with the Storybook each is drawn in.
 */
test('every story the tree declares is found where it stands', async () => {
  const declared = await stories()
  for (const [one, book] of [
    ['workspace--crowded', 'ui'],
    ['desktop-window--settings', 'ui'],
    ['flash-cards-window--cards-due', 'plugins'],
  ]) {
    assert.equal(declared.get(one), book, `${one} is declared in the ${book} Storybook`)
  }
})
