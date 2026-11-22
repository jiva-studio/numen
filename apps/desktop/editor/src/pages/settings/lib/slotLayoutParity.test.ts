import { describe, expect, it } from 'vitest'
import {
  collapseSeparators,
  createSlotRegistry,
  type DeclaredSlotEntry,
} from '@numen/plugin-loader'
import type { JsonValue } from '@/shared/api/json'
import { getSlotRows } from './slotLayout'

const SLOT = 'titleBar.commands'
const DRAWER = 'org.example.bar'
const entries: readonly DeclaredSlotEntry[] = [
  { pluginId: 'org.example.a', slot: SLOT, command: 'one', group: '0-navigate' },
  { pluginId: 'org.example.b', slot: SLOT, command: 'two', group: '1-new' },
  { pluginId: 'org.example.c', slot: SLOT, command: 'three', group: '2-settings' },
  { pluginId: 'org.example.d', slot: SLOT, command: 'four', group: '2-settings' },
]
const keyOf = (one: DeclaredSlotEntry) => `${one.pluginId}/command/${one.pluginId}:${one.command}`
const [one, two, three, four] = entries.map(keyOf) as [string, string, string, string]
const SEPARATOR = { id: '---' }

type Layout = Record<string, { id: string; hidden?: boolean }[]>

const drawn = (layout: Layout | undefined) =>
  createSlotRegistry({
    getEntries: () => entries,
    drawers: {
      getPoints: () => [{ pluginId: DRAWER, slot: SLOT, title: 'Buttons', menu: false }],
      getLayout: () => layout,
    },
    findCommand: (command) => ({ title: command }),
    findView: () => undefined,
  })
    .getEntries(SLOT)
    .map((item) => (item.kind === 'separator' ? '|' : item.key))

const listed = (layout: Layout | undefined) => {
  const contributors = entries.map((entry) => ({
    key: keyOf(entry),
    label: entry.command ?? '',
    pluginName: entry.pluginId,
    group: entry.group,
  }))
  const rows = getSlotRows(
    { slot: SLOT, title: 'Buttons', contributors },
    layout as unknown as JsonValue,
  )
  const shown = rows.filter((row) => row.kind === 'separator' || !row.isHidden)
  return collapseSeparators(shown, (row) => row.kind === 'separator').map((row) =>
    row.kind === 'separator' ? '|' : row.key,
  )
}

describe('the Slots tab and the slot', () => {
  it.each<[string, Layout | undefined]>([
    ['nothing stored', undefined],
    ['the last group first', { [SLOT]: [{ id: three }, SEPARATOR, { id: one }] }],
    [
      'a row moved over a separator',
      { [SLOT]: [{ id: one }, { id: two }, SEPARATOR, { id: three }] },
    ],
    [
      'a group left empty by hidden rows',
      { [SLOT]: [{ id: one }, SEPARATOR, { id: two, hidden: true }, SEPARATOR, { id: three }] },
    ],
    [
      'separators that open, close and repeat',
      { [SLOT]: [SEPARATOR, { id: four }, SEPARATOR, SEPARATOR, { id: one }, SEPARATOR] },
    ],
    [
      'a list with garbage in it',
      { [SLOT]: [{ id: 'gone' }, { id: two }, { id: two }, SEPARATOR] },
    ],
  ])('list the entries and separators in the order the slot draws them: %s', (_, layout) => {
    expect(listed(layout)).toStrictEqual(drawn(layout))
  })
})
