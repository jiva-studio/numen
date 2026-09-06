import { describe, expect, it } from 'vitest'

import { namesOf } from './naming'

/**
 * The reverse table is handed out as holding a key per word, and the cast that
 * says so checks nothing. A word with no key reads back as undefined wearing
 * the enum's type, and the window sends the schema a value it has no name for.
 */
describe('namesOf', () => {
  it('reads the sending side off the table of words', () => {
    const worded = { 0: null, 1: 'parent', 2: 'child' } as const
    expect(namesOf<'parent' | 'child', 0 | 1 | 2>(worded)).toEqual({ parent: 1, child: 2 })
  })

  it('refuses a table where two values answer to one word', () => {
    const worded = { 1: 'parent', 2: 'parent' } as const
    expect(() => namesOf<'parent', 1 | 2>(worded)).toThrow(/one word/)
  })
})
