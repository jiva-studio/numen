import assert from 'node:assert/strict'
import test from 'node:test'
import { sources } from './source.mjs'
import { baseline, declares, nouns, refused, words } from './verbs.mjs'

const declared = () => {
  const found = []
  for (const { at, text } of sources(['.ts', '.vue'])) {
    for (const name of declares(text)) found.push({ at, name })
  }
  return found
}

/**
 * A gerund or a participle names the doing of a thing; a function is named for
 * what it does when it is called. A word let through is a word written into the
 * dictionary with what it means.
 */
test('no function of the interface modules is named by a gerund or a participle', () => {
  const found = declared()
  const wrong = found
    .filter(({ name }) => refused(name) && !baseline.includes(name))
    .map(({ at, name }) => `${at} declares ${name}`)
  assert.deepEqual(wrong, [])

  // A walk that read no declaration is a rule checked against nothing, and it
  // passes. A count alone cannot say which modules it read, so the furthest of
  // them is named: the phone is where a rule stops at a border first.
  assert.ok(found.length > 800, `${found.length} functions read: the walk is not reading the modules`)
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
test('every word in the dictionary names something, and every baseline entry stands', () => {
  const found = declared()
  const said = new Set()
  for (const { name } of found) said.add(words(name)[0])
  assert.deepEqual(
    Object.keys(nouns).filter((one) => !said.has(one)),
    [],
  )

  const names = new Set(found.map((one) => one.name))
  assert.deepEqual(
    baseline.filter((one) => !names.has(one)),
    [],
  )
})

/**
 * What the rule refuses, read against names written to be refused. The first
 * word is the verb and the rest says what it acts on, so a name ending in a
 * plural is never this rule's business.
 */
test('what the verb rule refuses', () => {
  const cases = [
    { says: 'a gerund alone', allowed: false, name: 'dressing' },
    { says: 'a participle alone', allowed: false, name: 'offered' },
    { says: 'a gerund with what it acts on after it', allowed: false, name: 'mintingCardId' },
    { says: 'an imperative verb', allowed: true, name: 'getSettings' },
    { says: 'an imperative verb with a plural after it', allowed: true, name: 'resolveAddresses' },
    { says: 'a predicate', allowed: true, name: 'isRenaming' },
    { says: 'a composable', allowed: true, name: 'useDeckTabs' },
    { says: 'a handler', allowed: true, name: 'onDropEntries' },
    { says: 'a factory named by a plain noun', allowed: true, name: 'noteTitles' },
    { says: 'a word in the dictionary', allowed: true, name: 'settings' },
    // A word ending in those letters by accident. Two letters is the shortest
    // an English verb runs to, so one in front of the ending is not one.
    { says: 'a word that ends there by accident', allowed: true, name: 'bed' },
    // A third person verb reads exactly as a plural noun, and a factory here
    // may take a plain noun. A person reads those.
    { says: 'a third-person verb, which no machine can see', allowed: true, name: 'carries' },
  ]

  const wrong = cases.filter((one) => refused(one.name)).map((one) => one.says)
  const wanted = cases.filter((one) => !one.allowed).map((one) => one.says)
  assert.deepEqual(wrong, wanted)
})

/** The declarations the walk has to find, and what it must not read as one. */
test('what counts as a function', () => {
  const source = [
    'export function alpha() {}',
    'async function beta() {}',
    'const gamma = () => 1',
    'export const delta = async (one: string) => one',
    'const epsilon: Holder = (one) => one',
    'const zeta = (one = held()) => one',
    'const eta = 3',
    'class Eta { theta() {} }',
    'export interface Iota { kappa(): void }',
    ' * function lambda is named only in a comment',
    'const mu = `',
    'function nu() {}',
    '`',
  ].join('\n')
  assert.deepEqual(declares(source), ['alpha', 'beta', 'gamma', 'delta', 'epsilon', 'zeta'])
})
