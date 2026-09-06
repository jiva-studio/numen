import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import test from 'node:test'
import { root } from './source.mjs'
import { stories } from './reach.mjs'

const PREVIEW = 'modules/libs/ui/.storybook/preview.ts'

/**
 * Every story the interface library holds is walked with the keyboard once it
 * has been played, and what the walk finds is judged. The judging is a rule
 * only for as long as it is wired to every story: an `afterEach` in the
 * preview is what makes it every story rather than the handful that remembered
 * to ask.
 */
test('the keyboard walk runs after every story of the interface library', () => {
  const preview = readFileSync(join(root, PREVIEW), 'utf8')

  assert.ok(preview.includes('afterEach'), `${PREVIEW} runs nothing after a story`)
  assert.match(
    preview,
    /await walk\(\)/,
    `${PREVIEW} does not walk the keyboard through the story it has just drawn`,
  )
  assert.match(
    preview,
    /faults\(/,
    `${PREVIEW} walks the keyboard and makes nothing of what it finds`,
  )
})

/**
 * A rule that runs after every story covers nothing if there are no stories.
 * The count is a floor well under what the module holds, and the two files the
 * faults of this audit were found in are named: a walk that reads forty others
 * and not those is reading the wrong tree.
 */
test('there are stories for the keyboard walk to run after', () => {
  const found = stories().map((at) => relative(root, at))

  assert.ok(
    found.length > 30,
    `${found.length} story files found under the interface library: the walk is not reading them`,
  )
  for (const wanted of [
    'modules/libs/ui/src/workspace/Workspace.stories.ts',
    'modules/libs/ui/src/reader/Reader.stories.ts',
  ])
    assert.ok(found.includes(wanted), `no ${wanted}, which the keyboard rule was written for`)
})
