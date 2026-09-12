import assert from 'node:assert/strict'
import test from 'node:test'
import { sources } from './source.mjs'
import { asks, binds, packageOf } from './refs.mjs'

/**
 * A ref nothing binds is a reader that is always `null`. The window goes on,
 * draws everything, and one thing it offers does nothing at all.
 */
test('every template ref a window asks for is bound in its package', () => {
  const bound = new Map()
  const wanted = []
  for (const one of sources(['.ts', '.vue'])) {
    const held = bound.get(packageOf(one.at)) ?? new Set()
    for (const name of binds(one.text)) held.add(name)
    bound.set(packageOf(one.at), held)
    for (const name of asks(one.text)) wanted.push({ at: one.at, name })
  }

  const wrong = wanted
    .filter((one) => !bound.get(packageOf(one.at))?.has(one.name))
    .map((one) => `${one.at} asks for a ref nothing binds: ${one.name}`)
  assert.deepEqual(wrong, [])

  // A walk that read no ref is a rule checked against nothing, and it passes.
  // A count alone cannot say which packages it read, so one of each is named.
  assert.ok(wanted.length > 40, `${wanted.length} template refs read: the walk is not reading the windows`)
  for (const one of ['modules/libs/ui', 'modules/apps/desktop/editor']) {
    assert.ok(
      wanted.some((said) => said.at.startsWith(one)),
      `the walk did not read ${one}, so the rule stops at its border`,
    )
  }
})

/** What counts as asking for a ref, and what counts as binding one. */
test('what the template ref rule reads', () => {
  const held = [
    "const page = useTemplateRef<InstanceType<typeof NotesPanel>>('page')",
    "const box = useTemplateRef('box')",
    "// useTemplateRef('commented')",
    "const said = 'useTemplateRef(\\'quoted\\')'",
  ].join('\n')
  assert.deepEqual(asks(held), ['page', 'box'])

  const drawn = [
    '<template>',
    '  <NotesPanel ref="page" :held="one" />',
    '  <div :ref="whichever" />',
    '</template>',
  ].join('\n')
  assert.deepEqual(binds(drawn), ['page'])
})

/** A package is where a ref reaches: a composable is bound by its caller. */
test('the package a file stands in', () => {
  assert.equal(
    packageOf('modules/apps/desktop/flashcards/src/app/window.ts'),
    'modules/apps/desktop/flashcards',
  )
  assert.equal(packageOf('modules/libs/ui/src/features/plex/ui/Plex.vue'), 'modules/libs/ui')
})
