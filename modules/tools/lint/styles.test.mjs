import assert from 'node:assert/strict'
import test from 'node:test'
import { sources } from './source.mjs'
import { unreachable } from './styles.mjs'

/**
 * A component styles what its own template can put on an element, and nothing
 * else. A class left in the style block after the template stopped setting it
 * is not a dead rule: what hangs off it — a custom property a child reads, a
 * size a parent sets — stops arriving, and every story goes on passing,
 * because a story asserts an arrangement and never a size.
 */
test('no component styles a class no template of its can set', () => {
  const wrong = []
  const read = []
  for (const { at, text } of sources(['.vue'])) {
    if (!text.includes('<style')) continue
    read.push(at)
    for (const one of unreachable(text)) wrong.push(`${at} styles .${one}, which nothing sets`)
  }
  assert.deepEqual(wrong, [])

  // A walk that read no style block is a rule checked against nothing, and it
  // passes. The count is a floor well under what the modules hold, and the file
  // the rule was written for is named: a walk that reads eighty others and not
  // that one is reading the wrong tree.
  assert.ok(
    read.length > 60,
    `${read.length} components with a style block read: the walk is not reading them`,
  )
  assert.ok(
    read.some((at) => at.endsWith('welcome/WelcomePage.vue')),
    'the walk did not read WelcomePage.vue, which is the component this rule exists for',
  )
})

/**
 * What the rule refuses, read against blocks written to be refused. The three
 * families it has to let through are not exemptions: each is a class this
 * template does put on an element, by a route other than writing it out.
 */
test('what the style rule refuses', () => {
  const cases = [
    {
      says: 'a class the template sets',
      allowed: true,
      source: '<template><p class="note__body" /></template><style>.note__body { color: red }</style>',
    },
    {
      says: 'a class the template stopped setting',
      allowed: false,
      source: '<template><p class="note__text" /></template><style>.note__body { color: red }</style>',
    },
    {
      says: 'a class built by putting a value after a stem',
      allowed: true,
      source:
        '<template><p :class="`note__body--${tone}`" /></template><style>.note__body--warm {}</style>',
    },
    {
      says: 'a stage of a named transition, which Vue writes itself',
      allowed: true,
      source:
        '<template><Transition name="note__swap"><p /></Transition></template><style>.note__swap-enter-active {}</style>',
    },
    {
      says: "a class of the editor this component mounts, reached through :deep",
      allowed: true,
      source: '<template><div class="editor" /></template><style>.editor :deep(.cm-content) {}</style>',
    },
    {
      says: 'a class the style block only mentions in a comment',
      allowed: true,
      source: '<template><p /></template><style>/* .note__body is gone */ p { margin: 0 }</style>',
    },
    {
      says: 'a component with no style block of its own',
      allowed: true,
      source: '<template><p class="note__body" /></template>',
    },
  ]

  const refused = cases.filter((one) => unreachable(one.source).length > 0).map((one) => one.says)
  const wanted = cases.filter((one) => !one.allowed).map((one) => one.says)
  assert.deepEqual(refused, wanted)
})
