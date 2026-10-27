/**
 * What the menu on a row of the tree offers: what the tab does itself, among the
 * window's commands over the row.
 */
import { describe, expect, it } from 'vitest'
import { groupItems } from '@numen/ui'
import type { Entry } from '../types'
import { getPathCommands } from '../fixtures/host'
import { createMenuItems, NEW_FOLDER, NEW_NOTE, OWN_IDS, RENAME } from './menu'

const entry = (overrides: Partial<Entry> = {}): Entry => ({
  path: 'Entropy.md',
  name: 'Entropy.md',
  isFolder: false,
  isNote: true,
  type: 'note',
  ...overrides,
})

/** What the menu offers on one row, by the identity of each item, as it is drawn. */
const getGroupedIds = (overrides: Partial<Entry> = {}, isSeveral = false): readonly string[] =>
  groupItems([
    ...createMenuItems(getPathCommands({ entry: entry(overrides), isSeveral }), isSeveral),
  ]).map((one) => `${one.isRule ? '— ' : ''}${one.id}`)

const getMenu = (overrides: Partial<Entry> = {}, isSeveral = false) =>
  createMenuItems(getPathCommands({ entry: entry(overrides), isSeveral }), isSeveral)

describe('the menu on a row standing for a note', () => {
  it('opens it first, then what the tab does, then the rest of the window’s commands', () => {
    expect(getGroupedIds()).toStrictEqual([
      'read',
      'travel',
      '— new',
      'rename',
      'copy',
      '— links',
      '— ask',
      '— remove',
    ])

    const menu = getMenu()
    const newSubmenu = menu.find((one) => one.id === 'new')
    expect(newSubmenu?.children?.map((c) => c.id)).toStrictEqual([NEW_NOTE, NEW_FOLDER])

    const linksSubmenu = menu.find((one) => one.id === 'links')
    expect(linksSubmenu?.children?.map((c) => c.id)).toStrictEqual([
      'child',
      'parent',
      'jump',
      'title',
    ])
  })
})

describe('the menu on a row standing for any other file', () => {
  it('remains what the tab does before the window’s commands, a run after the path', () => {
    expect(getGroupedIds({ isNote: false, path: 'Lecture.mp3' })).toStrictEqual([
      'new',
      'rename',
      'copy',
      '— extract',
      'refine',
      '— remove',
    ])
  })

  it('leaves no rule behind on a row that has no run, and treats a folder as a file', () => {
    expect(getGroupedIds({ isNote: false, path: 'Cover.png' })).toStrictEqual([
      'new',
      'rename',
      'copy',
      '— remove',
    ])
    expect(getGroupedIds({ isFolder: true, isNote: false, path: 'physics' })).toStrictEqual([
      'new',
      'rename',
      'copy',
      '— remove',
    ])
  })
})

describe('the menu over a selection of several', () => {
  it('offers the window’s commands alone, which mean something for all of them', () => {
    expect(getGroupedIds({}, true)).toStrictEqual(['remove'])
  })
})

describe('the menu off every row', () => {
  it('offers what the tab makes at the root, and nothing to rename', () => {
    const items = createMenuItems(null, false)
    expect(items.map((one) => one.id)).toStrictEqual(['new'])
    expect(items[0]?.children?.map((one) => one.id)).toStrictEqual([NEW_NOTE, NEW_FOLDER])
  })
})

describe('what the tab does itself', () => {
  it('is the files it makes and the rename, and none of the window’s commands', () => {
    expect(OWN_IDS.has(NEW_NOTE)).toBe(true)
    expect(OWN_IDS.has(NEW_FOLDER)).toBe(true)
    expect(OWN_IDS.has(RENAME)).toBe(true)
    expect(OWN_IDS.has('remove')).toBe(false)
    expect(OWN_IDS.has('transcribe')).toBe(false)
  })
})
