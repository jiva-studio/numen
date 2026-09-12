import assert from 'node:assert/strict'
import test from 'node:test'
import { baseline, forced, quoted, refused } from './quotes.mjs'
import { sources } from './source.mjs'

/**
 * One dialect, so a file reads as this repository's. Nothing else settles it:
 * there is no formatter here, on purpose.
 */
test('every string of the interface modules is written in single quotes', () => {
  const files = sources(['.ts', '.vue'])
  const wrong = files.flatMap(({ at, text }) =>
    refused(at, text).map((one) => `${at}:${one.line} "${one.text}"`),
  )
  assert.deepEqual(wrong, [])

  // A walk that read no file finds no quotation mark and passes. The furthest
  // module is named, because a rule stops at a border first.
  assert.ok(files.length > 550, `${files.length} files read: the walk is not reading the modules`)
  assert.ok(
    files.some(({ at }) => at.endsWith('apps/mobile/src/core.ts')),
    "the walk did not read the phone's core.ts, so the rule stops at the mobile border",
  )

  // A line the rule reads wrongly is written down, and the day it reads it
  // rightly the line has to go. A baseline nobody can see the need for is a
  // rule kept alive by a line nobody reads.
  for (const one of baseline) {
    const [at, line] = one.split(':')
    const held = files.find((file) => file.at === at)
    assert.ok(held, `the baseline names ${at}, which the walk did not read`)
    assert.ok(
      quoted(held.text).some((said) => String(said.line) === line && !forced(said.text)),
      `the baseline names ${one}, where nothing is double-quoted any more`,
    )
  }
})

/** What the rule refuses, and what it has to let through. */
test('what the quote rule refuses', () => {
  const cases = [
    { says: 'a double-quoted string', allowed: false, source: 'const a = "one"' },
    { says: 'a double-quoted import', allowed: false, source: 'import a from "./one"' },
    { says: 'a single-quoted string', allowed: true, source: "const a = 'one'" },
    { says: 'a string holding the other quote', allowed: true, source: 'const a = "don\'t"' },
    { says: 'a template literal', allowed: true, source: 'const a = `one "two"`' },
    { says: 'a quotation mark in a line comment', allowed: true, source: '// says "one"' },
    { says: 'a quotation mark in a block comment', allowed: true, source: '/** says "one" */' },
  ]

  const wrong = cases.filter((one) => refused('a.ts', one.source).length > 0).map((one) => one.says)
  assert.deepEqual(
    wrong,
    cases.filter((one) => !one.allowed).map((one) => one.says),
  )
})

/** A component's template is HTML, where an attribute is written in double quotes. */
test("a component's template is not read, and its script is", () => {
  const source = [
    '<template>',
    '  <div class="row" :title="held">one</div>',
    '</template>',
    '',
    '<script setup lang="ts">',
    'const held = "one"',
    '</script>',
  ].join('\n')
  assert.deepEqual(
    refused('A.vue', source).map((one) => one.text),
    ['one'],
  )
})

/** Where a string stands, and what it holds, read off the source itself. */
test('what a double-quoted string is read as', () => {
  const source = ['const a = 1', 'const b = "two"'].join('\n')
  assert.deepEqual(quoted(source), [{ line: 2, text: 'two' }])
  assert.equal(forced("don't"), true)
  assert.equal(forced('two'), false)
})
