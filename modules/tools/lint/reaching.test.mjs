import assert from 'node:assert/strict'
import test from 'node:test'
import { depth, reached, refused } from './reaching.mjs'
import { sources } from './source.mjs'

/**
 * A path out of a slice counted in folders reads the same going down the layers
 * as going up them. Written through `@/` it says which layer it reached.
 */
test('no import of the window climbs out of its slice by counting folders', () => {
  const files = sources(['.ts', '.vue'])
  const wrong = files.flatMap(({ at, text }) => refused(at, text).map((one) => `${at} reaches ${one}`))
  assert.deepEqual(wrong, [])

  // A walk that read no file finds no path and passes, and this rule reads one
  // module: the file named is the deepest the window has.
  assert.ok(
    files.some(({ at }) => at.endsWith('editor/src/features/command-palette/model/navigation.ts')),
    'the walk did not read the palette, so the rule stops before the window',
  )
})

/** How far a file stands below what it may reach relatively. */
test('what a file may climb', () => {
  const at = 'modules/apps/desktop/editor/src/'
  assert.equal(depth(`${at}entities/tab/openers.ts`), 0)
  assert.equal(depth(`${at}entities/tab/@x/media.ts`), 1)
  assert.equal(depth(`${at}pages/preset-editor/ui/curve-slider/CurveSlider.vue`), 2)
  assert.equal(depth(`${at}shared/core.ts`), 0)
  assert.equal(depth(`${at}shared/words/errors.ts`), 1)
  assert.equal(depth(`${at}main.ts`), 0)
  assert.equal(depth('modules/libs/ui/src/features/plex/Plex.vue'), null)
})

/** What the rule refuses, and what it has to let through. */
test('what the reaching rule refuses', () => {
  const at = 'modules/apps/desktop/editor/src/entities/tab/@x/media.ts'
  const cases = [
    { says: 'a path out of the slice', allowed: false, source: "import { a } from '../../note'" },
    { says: 'a path to the slice root', allowed: true, source: "import { a } from '../openers'" },
    { says: 'a path beside it', allowed: true, source: "import { a } from './held'" },
    { says: 'a path through the alias', allowed: true, source: "import { a } from '@/entities/note'" },
    { says: 'a package', allowed: true, source: "import { a } from '@numen/ui'" },
  ]

  const wrong = cases.filter((one) => refused(at, one.source).length > 0).map((one) => one.says)
  assert.deepEqual(
    wrong,
    cases.filter((one) => !one.allowed).map((one) => one.says),
  )
})

/** The forms a path is written in, and how far each climbs. */
test('what counts as a path, and how far it goes', () => {
  const source = [
    "import { a } from '../../one'",
    "const b = await import('../two')",
    "vi.mock('../../../three')",
    "import { c } from '@/four'",
    "import { d } from 'five'",
    '/*',
    "from '../../../../six' is named only in a comment",
    '*/',
  ].join('\n')
  assert.deepEqual(
    reached(source).map((one) => `${one.said} ${one.up}`),
    ['../../one 2', '../two 1', '../../../three 3'],
  )
})
