import assert from 'node:assert/strict'
import test from 'node:test'
import { declares, nouns, owed, refused, words } from './nouns.mjs'
import { sources } from './source.mjs'

const declared = () => {
  const found = []
  for (const { at, text } of sources(['.ts', '.vue'])) {
    for (const name of declares(text)) found.push({ at, name })
  }
  return found
}

/**
 * A gerund or a participle says what is happening to a thing rather than what
 * the thing is. `Plexing` could not have been introduced without somebody
 * writing the word into the dictionary below and being asked what it means.
 */
test('no type of the interface modules is named by a gerund or a participle', () => {
  const found = declared()
  const wrong = found
    .filter(({ name }) => refused(name) && !owed.includes(name))
    .map(({ at, name }) => `${at} declares ${name}`)
  assert.deepEqual(wrong, [])

  // A walk that read no declaration is a rule checked against nothing, and it
  // passes. A count alone cannot say which modules it read, so the furthest of
  // them is named: the phone is where a rule stops at a border first.
  assert.ok(found.length > 400, `${found.length} types read: the walk is not reading the modules`)
  assert.ok(
    found.some(({ at }) => at.endsWith('apps/mobile/src/core.ts')),
    "the walk did not read the phone's core.ts, so the rule stops at the mobile border",
  )
})

/**
 * A word nothing is named by any more is a word nobody has to argue for, and a
 * dictionary that keeps them fills up until it reads as a census. Both lists
 * only shrink.
 */
test('every word in the dictionary names something, and every debt is still owed', () => {
  const found = declared()
  const said = new Set()
  for (const { name } of found) {
    if (/(ing|ed)$/.test(name)) said.add(words(name).at(-1))
  }
  assert.deepEqual(
    Object.keys(nouns).filter((one) => !said.has(one)),
    [],
  )

  const names = new Set(found.map((one) => one.name))
  assert.deepEqual(
    owed.filter((one) => !names.has(one)),
    [],
  )
})

/**
 * What the rule refuses, read against names written to be refused. The words
 * it lets through are the dictionary's, and a name that does not end in those
 * letters is never this rule's business however it reads.
 */
test('what the noun rule refuses', () => {
  const cases = [
    { says: 'a gerund', allowed: false, name: 'Plexing' },
    { says: 'a participle', allowed: false, name: 'Configured' },
    { says: 'a gerund with a qualifier in front', allowed: false, name: 'PlexFiling' },
    { says: 'a noun in the dictionary', allowed: true, name: 'Heading' },
    { says: 'a compound of one', allowed: true, name: 'OpenRecording' },
    { says: 'a plain noun', allowed: true, name: 'Vault' },
    { says: 'a noun whose stem is a verb', allowed: true, name: 'Editor' },
    // Drawn is the participle the rule names and this walk cannot see: an
    // irregular one has no ending to test for, and a machine that tried would
    // be reading English rather than a suffix. A person catches those.
    { says: 'an irregular participle, which has no ending', allowed: true, name: 'Drawn' },
  ]

  const wrong = cases.filter((one) => refused(one.name)).map((one) => one.says)
  const wanted = cases.filter((one) => !one.allowed).map((one) => one.says)
  assert.deepEqual(wrong, wanted)
})

/** The declarations the walk has to find, and the clauses it must not read. */
test('what counts as a declaration', () => {
  const source = [
    'export interface Alpha { a: 1 }',
    'type Beta = string',
    'export type { Gamma } from "./elsewhere"',
    'import type { Delta } from "./elsewhere"',
    'export abstract class Epsilon {}',
    'export enum Zeta { One }',
    ' * type Eta is named only in a comment',
    '/*',
    'type Plexing = 1',
    '*/',
    'const theta = `',
    'type Filing = 2',
    '`',
  ].join('\n')
  assert.deepEqual(declares(source), ['Alpha', 'Beta', 'Epsilon', 'Zeta'])
})
