/**
 * A string is written in single quotes.
 *
 * This repository has no formatter, so nothing else settles this, and a file in
 * the other dialect reads as somebody else's. A string holding a single quote of
 * its own takes double quotes: escaping it would be worse than the inconsistency.
 */
import { masked } from './catches.mjs'
import { blocks } from './source.mjs'

/**
 * Where every double-quoted string of a file stands, and what it holds.
 *
 * The mask blanks what a comment and a string say and leaves every other
 * character where it was, so a quotation mark inside either is nobody's
 * business and the offsets still read back onto the source.
 */
export function quoted(source) {
  const over = masked(source)
  const found = []
  for (const one of over.matchAll(/"[^"\n]*"/g)) {
    found.push({
      line: source.slice(0, one.index).split('\n').length,
      text: source.slice(one.index + 1, one.index + one[0].length - 1),
    })
  }
  return found
}

/**
 * baseline are the lines this rule reads wrongly, and the list only shrinks.
 *
 * The mask blanks a comment and a string and leaves a regular expression
 * standing — telling one from a division needs a parse `catches.mjs` exists to
 * do without — so a pattern matching an HTML attribute reads as two strings.
 * Single-quoting those would make the pattern match nothing.
 */
export const baseline = ['modules/apps/mobile/src/App.policy.test.ts:25']

/** Whether a string had to be written in double quotes: it holds the other one. */
export const forced = (text) => text.includes("'")

/**
 * The parts of a file this rule reads. A component's template is HTML, where an
 * attribute is written in double quotes; its script is ours.
 */
const read = (at, text) => (at.endsWith('.vue') ? blocks(text, 'script') : [text])

/** Every double-quoted string a file writes that a single-quoted one would hold. */
export const refused = (at, text) =>
  read(at, text).flatMap((part) =>
    quoted(part).filter((one) => !forced(one.text) && !baseline.includes(`${at}:${one.line}`)),
  )
