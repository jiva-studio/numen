/**
 * The menu on a row of the tree, drawn from the entries of the slot `fileTree.context`.
 */
import { describe, expect, it } from 'vitest'
import { groupItems } from '@numen/ui'
import { createMenuItems, findMenuEntry } from './menu'
import {
  ASK,
  COPY_PATH,
  NEW_MENU,
  OPEN,
  RENAME_ENTRY,
  separator,
  SKETCH,
  TRASH,
} from '../fixtures/host'

/** The items as the menu draws them, by id, a rule marked before the first item of a group. */
const getGroupedIds = (entries: Parameters<typeof createMenuItems>[0]): readonly string[] =>
  groupItems([...createMenuItems(entries)]).map((one) => `${one.isRule ? '— ' : ''}${one.id}`)

describe('the menu items of the entries of a slot', () => {
  it('draws a row per command and a rule at each separator', () => {
    expect(
      getGroupedIds([OPEN, separator(1), RENAME_ENTRY, COPY_PATH, separator(2), TRASH]),
    ).toStrictEqual([OPEN.key, `— ${RENAME_ENTRY.key}`, COPY_PATH.key, `— ${TRASH.key}`])
  })

  it('draws a command under its title', () => {
    expect(createMenuItems([RENAME_ENTRY])).toStrictEqual([
      { id: RENAME_ENTRY.key, text: 'Rename', group: '0' },
    ])
  })

  it('draws a submenu as a row that opens its own entries', () => {
    const [row] = createMenuItems([NEW_MENU])

    expect(row?.text).toBe('New')
    expect(row?.children?.map((one) => one.id)).toStrictEqual(
      NEW_MENU.entries.map((one) => one.key),
    )
  })

  it('draws a view as a row under the key of its entry', () => {
    expect(createMenuItems([ASK, separator(1), SKETCH]).map((one) => one.id)).toStrictEqual([
      ASK.key,
      SKETCH.key,
    ])
  })

  it('draws a view as a row holding its component, named by the id of the view', () => {
    expect(createMenuItems([SKETCH])).toStrictEqual([
      { id: SKETCH.key, text: 'sketch', group: '0', component: SKETCH.component },
    ])
  })

  it('draws nothing for a slot without entries', () => {
    expect(createMenuItems([])).toStrictEqual([])
  })
})

describe('the entry of a chosen item', () => {
  it('is found at the top of the menu and inside a submenu', () => {
    const entries = [OPEN, separator(1), NEW_MENU]
    const inside = NEW_MENU.entries[0]!

    expect(findMenuEntry(entries, OPEN.key)).toBe(OPEN)
    expect(findMenuEntry(entries, inside.key)).toBe(inside)
  })

  it('is nothing for an id no entry carries, and for a separator', () => {
    const entries = [OPEN, separator(1), TRASH]

    expect(findMenuEntry(entries, 'md.numen.unknown/command/none')).toBeUndefined()
    expect(findMenuEntry(entries, 'separator/1')).toBeUndefined()
  })
})
