/**
 * The menu on a node, and what it offers.
 *
 * What it offers is every command over a note that has a row of its own, which
 * is the same list the palette draws over one, standing in groups.
 */
import { describe, expect, it } from 'vitest'

import { ITEMS, NEW_NOTE, NONE, OFFERED } from './menu'
import { commandsOf, overNote } from '../../features/command-palette/commands'
import { WORDS as words } from '../../shared/words'

describe('what the menu offers', () => {
  it('offers every command over a note, and nothing the palette does not', () => {
    const offered = overNote(commandsOf(words)).map((one) => one.id)
    expect([...ITEMS.map((item) => item.id)].sort()).toStrictEqual([...offered].sort())
  })

  it('says what each of them does, in the words the palette says it in', () => {
    const said = new Map(overNote(commandsOf(words)).map((one) => [one.id, one.text]))
    for (const item of ITEMS) {
      expect(item.text).not.toBe('')
      expect(item.text).toBe(said.get(item.id))
    }
  })

  it('stands the items in groups, each group whole and unbroken', () => {
    const groups = ITEMS.map((item) => item.group)
    for (const item of ITEMS) expect(item.group).not.toBeUndefined()
    expect([...new Set(groups)]).toStrictEqual(['open', 'file', 'plex', 'agent', 'remove'])
    // A group standing in two places would draw a rule through the middle of it.
    expect(groups.filter((group, at) => group !== groups[at - 1])).toStrictEqual([
      ...new Set(groups),
    ])
  })

  it('holds every item it draws, and no command reached on another one’s row', () => {
    for (const item of ITEMS) expect(OFFERED.has(item.id)).toBe(true)

    expect(OFFERED.has('destroy')).toBe(false)
    expect(OFFERED.has('constructor')).toBe(false)
  })
})

describe('what the menu off every node offers', () => {
  it('offers a note to be made, and says so', () => {
    expect(NONE.map((item) => item.id)).toStrictEqual([NEW_NOTE])
    for (const item of NONE) expect(item.text).not.toBe('')
  })

  it('offers no command over a note, there being no note it would be over', () => {
    for (const item of NONE) expect(OFFERED.has(item.id)).toBe(false)
  })
})
