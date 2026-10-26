/**
 * The menu on a node, and what it offers.
 */
import { describe, expect, it } from 'vitest'
import { createNodeItems, ITEMS, NEW_NOTE, NONE, OFFERED } from './menu'

describe('what the menu offers', () => {
  it('offers the commands of the window over a note', () => {
    const expected = ['read', 'copy', 'child', 'parent', 'jump', 'title', 'remove']
    expect([...ITEMS.map((item) => item.id)].sort()).toStrictEqual([...expected].sort())
  })

  it('says what each of them does', () => {
    for (const item of ITEMS) {
      expect(item.text).not.toBe('')
    }
  })

  it('remains the items in groups, each group whole and unbroken', () => {
    const groups = ITEMS.map((item) => item.group)
    for (const item of ITEMS) expect(item.group).not.toBeUndefined()
    expect([...new Set(groups)]).toStrictEqual(['open', 'file', 'plex', 'remove'])
    expect(groups.filter((group, index) => group !== groups[index - 1])).toStrictEqual([
      ...new Set(groups),
    ])
  })

  it('holds every item it draws, and no command reached on another row', () => {
    for (const item of ITEMS) expect(OFFERED.has(item.id)).toBe(true)

    expect(OFFERED.has('destroy')).toBe(false)
    expect(OFFERED.has('constructor')).toBe(false)
  })
})

describe('what the menu on a node offers of the commands plugins declare', () => {
  it('draws them after what opens the note, in a band of their own', () => {
    const items = createNodeItems([
      { id: 'plugin:share', text: 'Share the note' },
      { id: 'plugin:print', text: 'Print the note' },
    ])

    expect(items.map((item) => [item.id, item.group])).toStrictEqual([
      ['read', 'open'],
      ['plugin:share', 'declared'],
      ['plugin:print', 'declared'],
      ['copy', 'file'],
      ['child', 'plex'],
      ['parent', 'plex'],
      ['jump', 'plex'],
      ['title', 'plex'],
      ['remove', 'remove'],
    ])
  })

  it('is the commands of the window alone where no plugin declares one', () => {
    expect(createNodeItems([])).toStrictEqual(ITEMS)
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
