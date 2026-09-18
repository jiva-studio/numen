import assert from 'node:assert/strict'
import test from 'node:test'
import { goSources, sources } from './source.mjs'
import { baseline, declaresBooleans, goDeclaresBooleans, saysBoolean } from './booleans.mjs'

/** Every boolean field of the repository, under the file it stands in. */
const declared = () => {
  const found = []
  for (const { at, text } of goSources()) {
    if (at.includes('/gen/')) continue
    for (const name of goDeclaresBooleans(text)) found.push({ at, name })
  }
  for (const { at, text } of sources(['.ts', '.vue'])) {
    for (const name of declaresBooleans(text)) found.push({ at, name })
  }
  return found
}

/**
 * A reader meets a boolean by its name long before they meet its type. `stale`
 * and `disabled` read as a state somebody is in; `isStale` and `isDisabled` say
 * that what stands there is true or false.
 */
test('every boolean field says so in its name', () => {
  const wrong = declared()
    .filter(({ name }) => !saysBoolean(name))
    .map(({ at, name }) => `${at} declares ${name}`)
    .filter((one) => !baseline.includes(one))
  assert.deepEqual(wrong, [])
})

/**
 * A field that has been renamed or deleted is a field nobody has to argue for.
 * The baseline only shrinks, and an entry naming a field nothing declares any
 * more is an entry that stayed behind.
 */
test('every baseline entry names a field that is still there', () => {
  const said = new Set(declared().map(({ at, name }) => `${at} declares ${name}`))
  assert.deepEqual(
    baseline.filter((one) => !said.has(one)),
    [],
  )
})

/** What the rule reads as a boolean's name, and what it does not. */
test('what the boolean rule refuses', () => {
  const cases = [
    { says: 'a participle', allowed: false, name: 'changed' },
    { says: 'an adjective', allowed: false, name: 'stale' },
    // A prop bound straight to the attribute of the same name is named as the
    // platform named it, or the two stop lining up.
    { says: "an element's own attribute", allowed: true, name: 'disabled' },
    { says: 'a gerund', allowed: false, name: 'waiting' },
    { says: 'the three words of the rule', allowed: true, name: 'isStale' },
    { says: 'has', allowed: true, name: 'hasTranscript' },
    { says: 'can', allowed: true, name: 'canRead' },
    // A field matched against what a browser event carries is named as the
    // browser named it, or the two no longer line up.
    { says: "a browser event's own field", allowed: true, name: 'ctrlKey' },
  ]

  const wrong = cases.filter((one) => !saysBoolean(one.name)).map((one) => one.says)
  assert.deepEqual(wrong, cases.filter((one) => !one.allowed).map((one) => one.says))
})
