/**
 * The menu on a node, and what it offers.
 *
 * What it offers is every command over a note that has a row of its own, which
 * is the same list the palette draws over one.
 */
import { describe, expect, it } from 'vitest'

import { ITEMS, OFFERED } from './menu'
import { commandsOf, overNote } from '../commanding'
import { WORDS as words } from '../words'

describe('what the menu offers', () => {
  it('offers every command over a note, in the order the commands are drawn', () => {
    expect(ITEMS.map((item) => item.id)).toStrictEqual(
      overNote(commandsOf(words)).map((one) => one.id),
    )
  })

  it('says what each of them does', () => {
    for (const item of ITEMS) expect(item.text).not.toBe('')
  })

  it('holds every item it draws, and no command reached on another one’s row', () => {
    for (const item of ITEMS) expect(OFFERED.has(item.id)).toBe(true)

    expect(OFFERED.has('destroy')).toBe(false)
    expect(OFFERED.has('constructor')).toBe(false)
  })
})
