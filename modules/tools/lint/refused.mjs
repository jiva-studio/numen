/**
 * What the verb rule refuses right now, as a list a person can work through.
 *
 * Run with a path in front of it to read one tree: `node refused.mjs core/usecase`.
 * With `--exported` it says only the names that travel out of their package.
 */
import { goSources, sources } from './source.mjs'
import { declares, goDeclares, refused } from './verbs.mjs'

const [under = '', ...flags] = process.argv.slice(2)
const only = flags.includes('--exported')

const found = []
for (const { at, text } of goSources()) {
  for (const name of goDeclares(text)) found.push({ at, name })
}
for (const { at, text } of sources(['.ts', '.vue'])) {
  for (const name of declares(text)) found.push({ at, name })
}

const wrong = found.filter(
  (one) =>
    refused(one.name) &&
    one.at.includes(under) &&
    (!only || /^[A-Z]/.test(one.name)),
)
for (const one of wrong) console.log(`${one.at}\t${one.name}`)
console.log(`\n${wrong.length} names`)
