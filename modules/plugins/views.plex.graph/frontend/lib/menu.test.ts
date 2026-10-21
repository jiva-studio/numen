/**
 * The menu on a node, and what it offers.
 */
import { describe, expect, it } from 'vitest'
import { ITEMS, NEW_NOTE, NONE, OFFERED } from './menu'

describe('what the menu offers', () => {
  it('offers commands over a note', () => {
    const expected = [
      'read',
      'travel',
      'preset',
      'copy',
      'reveal',
      'child',
      'parent',
      'jump',
      'title',
      'ask',
      'remove',
    ]
    expect([...ITEMS.map((item) => item.id)].sort()).toStrictEqual([...expected].sort())
  })

  it('says what each of them does', () => {
    for (const item of ITEMS) {
      expect(item.text).not.toBe('')
    }
  })

  it('stands the items in groups, each group whole and unbroken', () => {
    const groups = ITEMS.map((item) => item.group)
    for (const item of ITEMS) expect(item.group).not.toBeUndefined()
    expect([...new Set(groups)]).toStrictEqual(['open', 'file', 'plex', 'agent', 'remove'])
    expect(groups.filter((group, at) => group !== groups[at - 1])).toStrictEqual([
      ...new Set(groups),
    ])
  })

  it('holds every item it draws, and no command reached on another row', () => {
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
