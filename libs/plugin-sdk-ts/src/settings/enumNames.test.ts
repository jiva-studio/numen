import { describe, expect, it } from 'vitest'

import { getNames } from './enumNames'

/**
 * The reverse table is typed as holding a key per word, and the cast asserting
 * that checks nothing. A word with no key reads back as undefined with the
 * enum's type, and the window sends the schema a value it has no name for.
 */
describe('getNames', () => {
  it('reads the sending side off the table of words', () => {
    const worded = { 0: null, 1: 'parent', 2: 'child' } as const
    expect(getNames<'parent' | 'child', 0 | 1 | 2>(worded)).toEqual({ parent: 1, child: 2 })
  })

  it('refuses a table where two values answer to one word', () => {
    const worded = { 1: 'parent', 2: 'parent' } as const
    expect(() => getNames<'parent', 1 | 2>(worded)).toThrow(/one word/)
  })
})
