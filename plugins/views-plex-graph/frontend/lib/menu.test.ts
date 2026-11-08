/**
 * The menu off every node, and what it offers.
 */
import { describe, expect, it } from 'vitest'
import { NEW_NOTE, NONE } from './menu'

describe('what the menu off every node offers', () => {
  it('offers a note to be made, and says so', () => {
    expect(NONE.map((item) => item.id)).toStrictEqual([NEW_NOTE])
    for (const item of NONE) expect(item.text).not.toBe('')
  })
})
