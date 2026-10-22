/**
 * What the menu on a row of the tree offers: what the tab does itself, among the
 * window's commands over the row.
 */
import { describe, expect, it } from 'vitest'
import { groupItems } from '@numen/ui'
import type { Entry } from '../types'
import { getPathCommands } from '../fixtures/host'
import {
  createMenuItems,
  NEW_DECK,
  NEW_FOLDER,
  NEW_NOTE,
  NEW_PRESET,
  NEW_STENCIL,
  OWN_IDS,
  RENAME,
} from './menu'

const entry = (overrides: Partial<Entry> = {}): Entry => ({
  path: 'Entropy.md',
  name: 'Entropy.md',
  isFolder: false,
  kind: 'note',
  type: 'note',
  ...overrides,
})

/** What the menu offers on one row, by the identity of each item, as it is drawn. */
const getGroupedIds = (overrides: Partial<Entry> = {}, isSeveral = false): readonly string[] =>
  groupItems([
    ...createMenuItems(getPathCommands({ entry: entry(overrides), isSeveral }), isSeveral),
  ]).map((one) => `${one.isRule ? '— ' : ''}${one.id}`)

const OWN = [NEW_NOTE, NEW_DECK, NEW_STENCIL, NEW_PRESET, NEW_FOLDER, RENAME]

describe('the menu on a row standing for a note', () => {
  it('opens it first, then what the tab does, then the rest of the window’s commands', () => {
    expect(getGroupedIds()).toStrictEqual([
      'read',
      'travel',
      `— ${NEW_NOTE}`,
      ...OWN.slice(1),
      'copy',
      '— child',
      'parent',
      'jump',
      'title',
      '— ask',
      '— remove',
    ])
  })
})

describe('the menu on a row standing for any other file', () => {
  it('remains what the tab does before the window’s commands, a run after the path', () => {
    expect(getGroupedIds({ kind: 'recording', path: 'Lecture.mp3' })).toStrictEqual([
      ...OWN,
      'copy',
      '— transcribe',
      'proofread',
      '— remove',
    ])
  })

  it('leaves no rule behind on a row that has no run, and treats a folder as a file', () => {
    expect(getGroupedIds({ kind: 'other', path: 'Cover.png' })).toStrictEqual([
      ...OWN,
      'copy',
      '— remove',
    ])
    expect(getGroupedIds({ isFolder: true, kind: 'other', path: 'physics' })).toStrictEqual([
      ...OWN,
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
    expect(createMenuItems(null, false).map((one) => one.id)).toStrictEqual([
      NEW_NOTE,
      NEW_DECK,
      NEW_STENCIL,
      NEW_PRESET,
      NEW_FOLDER,
    ])
  })
})

describe('what the tab does itself', () => {
  it('is the files it makes and the rename, and none of the window’s commands', () => {
    expect([...OWN_IDS]).toStrictEqual(OWN)
    expect(OWN_IDS.has('remove')).toBe(false)
    expect(OWN_IDS.has('transcribe')).toBe(false)
  })
})
