import assert from 'node:assert/strict'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { corpora, idOf, unproved, unshared, unstaged, unwalked } from './reach.mjs'

/**
 * Every package that runs stories walks them. The judging is a rule only for
 * as long as it is wired to every story of every package that draws one: an
 * `afterEach` in a preview is what makes it every story, and a package whose
 * preview holds no walk is a corner of the tree the audit reports as clean
 * without having looked at it.
 */
test('every package that runs stories walks them', () => {
  for (const one of corpora()) assert.deepEqual(unwalked(one), [])
})

/**
 * The walk is one walk. A preview reaching it by name is a preview that cannot
 * drift from it; a preview with a walk of its own is a second rule to keep in
 * step with the first, and the two diverge the week after they are written.
 */
test('there is one keyboard walk, and every preview reaches it', () => {
  assert.deepEqual(unshared(corpora()), [])
})

/**
 * A walk that finds nothing passes everything, so each package names one story
 * of its own and the fewest stops the keyboard finds there. A floor named
 * against another package's story proves nothing about this one, and a floor
 * named against a story that does not exist proves nothing about any.
 */
test('each walk is proved against a story of its own package', () => {
  for (const one of corpora()) assert.deepEqual(unproved(one), [])
})

/**
 * A rule that runs after every story covers nothing if there are no stories.
 * The counts are floors well under what each package holds, and the files the
 * faults of this audit were found in are named: a walk that reads forty others
 * and not those is reading the wrong tree.
 */
test('there are stories for the keyboard walk to run after', () => {
  const wanted = {
    '@numen/ui': [
      30,
      [
        'modules/libs/ui/src/features/workspace/WorkspaceLayout.stories.ts',
        'modules/libs/ui/src/features/reader/Reader.stories.ts',
        'modules/libs/ui/src/features/book/Book.stories.ts',
      ],
    ],
    '@numen/editor': [
      5,
      [
        'modules/apps/desktop/editor/src/app/screens.stories.ts',
        'modules/apps/desktop/editor/src/pages/settings/ui/setting-row/SettingRow.stories.ts',
      ],
    ],
    '@numen/flashcards': [1, ['modules/apps/desktop/flashcards/src/screens.stories.ts']],
  }
  assert.deepEqual(unstaged(corpora(), wanted), [])
})

/**
 * What the four rules refuse, read against a package written to be refused:
 * one story, a preview that runs something else after it, proves its walk
 * against a story it does not hold, and takes no walk from a check that walks
 * nothing.
 */
test('what the reach rules refuse', () => {
  const under = mkdtempSync(join(tmpdir(), 'reach-'))
  try {
    mkdirSync(join(under, 'scratch/src'), { recursive: true })
    mkdirSync(join(under, 'scratch/.storybook'), { recursive: true })
    writeFileSync(
      join(under, 'scratch/src/Thing.stories.ts'),
      ["const meta = {", "  title: 'Thing',", '}', 'export default meta', 'export const Plain = {}', ''].join('\n'),
    )
    writeFileSync(
      join(under, 'scratch/.storybook/preview.ts'),
      ['export default {', '  afterEach: () => {},', "  parameters: { story: 'thing--other', stops: 3 },", '}', ''].join('\n'),
    )
    writeFileSync(join(under, 'check.ts'), 'export const reachCheck = () => async () => {}\n')

    const found = corpora([{ name: 'scratch', at: 'scratch' }], under)
    assert.equal(found.length, 1)
    const [one] = found
    assert.deepEqual(one.stories, ['scratch/src/Thing.stories.ts'])
    assert.deepEqual([...one.ids], ['thing--plain'])

    const preview = 'scratch/.storybook/preview.ts'
    assert.deepEqual(unwalked(one, under), [
      `${preview} does not walk the keyboard through the story it has just drawn`,
    ])
    assert.deepEqual(unshared(found, under, 'check.ts'), [
      'check.ts does not walk the keyboard',
      'check.ts walks the keyboard and makes nothing of what it finds',
      `${preview} does not take the walk from check.ts`,
    ])
    assert.deepEqual(unproved(one, under), [
      `${preview} is proved against thing--other, which is no story of scratch`,
    ])
    assert.deepEqual(
      unstaged(found, { scratch: [2, ['scratch/src/Thing.stories.ts', 'scratch/src/Other.stories.ts']] }),
      [
        '1 story files found under scratch: the walk is not reading them',
        'no scratch/src/Other.stories.ts, which the keyboard rule was written for',
      ],
    )
  } finally {
    rmSync(under, { recursive: true, force: true })
  }
})

/** The addresses the proofs are checked against are the ones Storybook gives. */
test('a story is addressed as Storybook addresses it', () => {
  assert.equal(idOf('Workspace', 'Crowded'), 'workspace--crowded')
  assert.equal(idOf('Flash Cards/Window', 'CardsDue'), 'flash-cards-window--cards-due')
  assert.equal(idOf('Window/Setting row', 'EveryControlAtTheOneEdge'), 'window-setting-row--every-control-at-the-one-edge')
})
