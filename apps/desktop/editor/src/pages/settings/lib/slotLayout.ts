import { arrangeByLayout, insertSeparators, SLOT_SEPARATOR_ID } from '@numen/plugin-loader'
import type { JsonValue } from '@/shared/api/json'
import type { SlotContributor, SlotGroup } from '../types'

/** One entry of a slot as the settings page draws it, in the order the slot draws it. */
interface EntryRow {
  readonly kind: 'entry'
  readonly key: string
  readonly label: string
  readonly pluginName: string
  readonly isHidden: boolean
}

/** The line the slot draws between two groups of its entries. */
interface SeparatorRow {
  readonly kind: 'separator'
  readonly key: string
}

export type SlotRow = EntryRow | SeparatorRow

type StoredEntry = { readonly id: string; readonly hidden?: boolean }

const isRecord = (value: JsonValue | undefined): value is { readonly [key: string]: JsonValue } =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const isStoredEntry = (value: JsonValue): value is StoredEntry =>
  isRecord(value) && typeof value['id'] === 'string'

function getStoredEntries(layout: JsonValue | undefined, slot: string): readonly StoredEntry[] {
  if (!isRecord(layout)) return []
  const list = layout[slot]
  return Array.isArray(list) ? list.filter(isStoredEntry) : []
}

const getDefaults = (contributors: readonly SlotContributor[]): (SlotContributor | null)[] =>
  insertSeparators<SlotContributor, SlotContributor | null>(contributors, {
    group: (one) => one.group,
    buildRow: (one) => one,
    buildSeparator: () => null,
  })

/**
 * The rows of a slot: its entries and the separators between groups, as the stored layout arranges them,
 * else as the manifests do. Every stored separator is a row, even one the bar does not draw because it
 * opens the slot, closes it, follows another or has only hidden entries around it.
 */
export function getSlotRows(group: SlotGroup, layout: JsonValue | undefined): readonly SlotRow[] {
  const stored = getStoredEntries(layout, group.slot)
  const hidden = new Set(stored.filter((one) => one.hidden === true).map((one) => one.id))
  const arranged = arrangeByLayout(getDefaults(group.contributors), (one) => one.key, stored)
  let separators = 0
  const rows = arranged.map((one): SlotRow =>
    one === null
      ? { kind: 'separator', key: createSeparatorKey(separators++) }
      : { kind: 'entry', ...one, isHidden: hidden.has(one.key) },
  )
  return rows
}

const createSeparatorKey = (ordinal: number): string => `separator:${ordinal}`

function assignSeparatorKeys(rows: readonly SlotRow[]): readonly SlotRow[] {
  let ordinal = 0
  return rows.map((one) =>
    one.kind === 'separator' ? { kind: 'separator', key: createSeparatorKey(ordinal++) } : one,
  )
}

/** Where the separator at `at` stands among the separators of the rows, counted from 1. */
export function getSeparatorPlace(
  rows: readonly SlotRow[],
  at: number,
): { readonly position: number; readonly total: number } {
  const position = rows.slice(0, at + 1).filter((one) => one.kind === 'separator').length
  return { position, total: rows.filter((one) => one.kind === 'separator').length }
}

/**
 * The rows with the one at `from` moved to `to`; the rows as they were for a place outside the list.
 * The separators are keyed as the rows read back from the stored list are, so each row has the key it will have there.
 */
export function moveSlotRow(
  rows: readonly SlotRow[],
  from: number,
  to: number,
): readonly SlotRow[] {
  const moved = rows[from]
  if (!moved || from === to || to < 0 || to >= rows.length) return rows
  const next = rows.filter((_, at) => at !== from)
  next.splice(to, 0, moved)
  return assignSeparatorKeys(next)
}

/** The rows with the visibility of the entry at `at` turned over. */
export function toggleSlotRow(rows: readonly SlotRow[], at: number): readonly SlotRow[] {
  return rows.map((one, index) =>
    index === at && one.kind === 'entry' ? { ...one, isHidden: !one.isHidden } : one,
  )
}

/** The rows with a separator placed after the first row, where a person finds it to drag it on. */
export function addSlotSeparator(rows: readonly SlotRow[]): readonly SlotRow[] {
  if (rows.length === 0) return rows
  const used = new Set(rows.map((one) => one.key))
  let n = 0
  while (used.has(`separator:${n}`)) n += 1
  return [rows[0] as SlotRow, { kind: 'separator', key: `separator:${n}` }, ...rows.slice(1)]
}

/** The rows without the separator at `at`; the rows as they were for an entry. */
export function removeSlotSeparator(rows: readonly SlotRow[], at: number): readonly SlotRow[] {
  return rows[at]?.kind === 'separator' ? rows.filter((_, index) => index !== at) : rows
}

/** Whether the layout keeps a list for the slot. */
export function hasSlotLayout(layout: JsonValue | undefined, slot: string): boolean {
  return isRecord(layout) && Array.isArray(layout[slot])
}

/** The layout without the list of one slot, the other slots as they were, so the slot goes back to its manifests. */
export function resetSlotRows(layout: JsonValue | undefined, slot: string): JsonValue {
  const { [slot]: _dropped, ...others } = isRecord(layout) ? layout : {}
  return others
}

/** The layout with the entries of one slot replaced by the rows, the other slots as they were. */
export function writeSlotRows(
  layout: JsonValue | undefined,
  slot: string,
  rows: readonly SlotRow[],
): JsonValue {
  const others = isRecord(layout) ? layout : {}
  return {
    ...others,
    [slot]: rows.map((one) =>
      one.kind === 'separator'
        ? { id: SLOT_SEPARATOR_ID }
        : { id: one.key, ...(one.isHidden ? { hidden: true } : {}) },
    ),
  }
}
