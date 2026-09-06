import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import test from 'node:test'
import { root } from './source.mjs'
import { corpora, idOf } from './reach.mjs'

/** Where the one walk is written, and what a preview reaches it by. */
const CHECK = 'modules/libs/ui/.storybook/check.ts'

const read = (at) => readFileSync(join(root, at), 'utf8')

/**
 * Every package that runs stories walks them. The judging is a rule only for
 * as long as it is wired to every story of every package that draws one: an
 * `afterEach` in a preview is what makes it every story rather than the handful
 * that remembered to ask, and a package whose preview holds no walk is a corner
 * of the tree the audit reports as clean without having looked at it.
 */
test('every package that runs stories walks them', () => {
  for (const { name, preview, stories } of corpora()) {
    assert.ok(existsSync(join(root, preview)), `${name} draws ${stories.length} stories and has no ${preview}`)
    const text = read(preview)
    assert.match(text, /afterEach/, `${preview} runs nothing after a story`)
    assert.match(
      text,
      /afterEach:\s*reachCheck\(/,
      `${preview} does not walk the keyboard through the story it has just drawn`,
    )
  }
})

/**
 * The walk is one walk. A preview reaching it by name is a preview that cannot
 * drift from it; a preview with a walk of its own is a second rule to keep in
 * step with the first, and the two diverge the week after they are written.
 */
test('there is one keyboard walk, and every preview reaches it', () => {
  const check = read(CHECK)
  assert.match(check, /await walk\(\)/, `${CHECK} does not walk the keyboard`)
  assert.match(check, /faults\(/, `${CHECK} walks the keyboard and makes nothing of what it finds`)

  for (const { preview } of corpora()) {
    const to = relative(join(root, preview, '..'), join(root, CHECK)).replace(/\.ts$/, '')
    const at = to.startsWith('.') ? to : `./${to}`
    assert.ok(
      read(preview).includes(`from '${at}'`),
      `${preview} does not take the walk from ${CHECK}`,
    )
  }
})

/**
 * A walk that finds nothing passes everything, so each package names one story
 * of its own and the fewest stops the keyboard finds there. A floor named
 * against another package's story proves nothing about this one, and a floor
 * named against a story that does not exist proves nothing about any.
 */
test('each walk is proved against a story of its own package', () => {
  for (const { name, preview, ids } of corpora()) {
    const proof = /story:\s*'([^']+)',\s*stops:\s*(\d+)/.exec(read(preview))
    assert.ok(proof, `${preview} names no story to prove its walk against`)
    assert.ok(
      ids.has(proof[1]),
      `${preview} is proved against ${proof[1]}, which is no story of ${name}`,
    )
    assert.ok(Number(proof[2]) > 0, `${preview} asks its walk to find no stops, which anything does`)
  }
})

/**
 * A rule that runs after every story covers nothing if there are no stories.
 * The counts are floors well under what each package holds, and the files the
 * faults of this audit were found in are named: a walk that reads forty others
 * and not those is reading the wrong tree.
 */
test('there are stories for the keyboard walk to run after', () => {
  const held = new Map(corpora().map((one) => [one.name, one.stories]))

  const wanted = {
    '@numen/ui': [
      30,
      ['modules/libs/ui/src/workspace/Workspace.stories.ts', 'modules/libs/ui/src/reader/Reader.stories.ts'],
    ],
    '@numen/editor': [
      5,
      [
        'modules/apps/desktop/editor/src/screens.stories.ts',
        'modules/apps/desktop/editor/src/settings/controls/SettingRow.stories.ts',
      ],
    ],
    '@numen/flashcards': [1, ['modules/apps/desktop/flashcards/src/screens.stories.ts']],
  }

  for (const [name, [least, named]] of Object.entries(wanted)) {
    const found = held.get(name) ?? []
    assert.ok(
      found.length >= least,
      `${found.length} story files found under ${name}: the walk is not reading them`,
    )
    for (const one of named)
      assert.ok(found.includes(one), `no ${one}, which the keyboard rule was written for`)
  }
})

/** The addresses the proofs are checked against are the ones Storybook gives. */
test('a story is addressed as Storybook addresses it', () => {
  assert.equal(idOf('Workspace', 'Crowded'), 'workspace--crowded')
  assert.equal(idOf('Flash Cards/Window', 'CardsDue'), 'flash-cards-window--cards-due')
  assert.equal(idOf('Window/Setting row', 'EveryControlAtTheOneEdge'), 'window-setting-row--every-control-at-the-one-edge')
})
