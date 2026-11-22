import { arrangeByLayout, collapseSeparators, SLOT_SEPARATOR_ID } from '@numen/plugin-loader'
import type { JsonValue } from '@/shared/api/json'
import type { SlotContributor, SlotGroup } from '../types'

/** One entry of a slot as the settings page draws it, in the order the slot draws it. */
export interface EntryRow {
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

function getDefaults(contributors: readonly SlotContributor[]): (SlotContributor | null)[] {
  return contributors.flatMap((one, at) =>
    at > 0 && contributors[at - 1]?.group !== one.group ? [null, one] : [one],
  )
}

const isSeparator = (row: SlotRow): boolean => row.kind === 'separator'

/** The rows of a slot: its entries and the separators between groups, as the stored layout arranges them, else as the manifests do. */
export function getSlotRows(group: SlotGroup, layout: JsonValue | undefined): readonly SlotRow[] {
  const stored = getStoredEntries(layout, group.slot)
  const hidden = new Set(stored.filter((one) => one.hidden === true).map((one) => one.id))
  const arranged = arrangeByLayout(getDefaults(group.contributors), (one) => one.key, stored)
  let separators = 0
  const rows = arranged.map((one): SlotRow =>
    one === null
      ? { kind: 'separator', key: `separator:${separators++}` }
      : { kind: 'entry', ...one, isHidden: hidden.has(one.key) },
  )
  return collapseSeparators(rows, isSeparator)
}

/** The rows with the one at `from` moved to `to`; the rows as they were for a place outside the list. */
export function moveSlotRow(
  rows: readonly SlotRow[],
  from: number,
  to: number,
): readonly SlotRow[] {
  const moved = rows[from]
  if (!moved || from === to || to < 0 || to >= rows.length) return rows
  const next = rows.filter((_, at) => at !== from)
  next.splice(to, 0, moved)
  return next
}

/** The rows with the visibility of the entry at `at` turned over. */
export function toggleSlotRow(rows: readonly SlotRow[], at: number): readonly SlotRow[] {
  return rows.map((one, index) =>
    index === at && one.kind === 'entry' ? { ...one, isHidden: !one.isHidden } : one,
  )
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
    [slot]: collapseSeparators(rows, isSeparator).map((one) =>
      one.kind === 'separator'
        ? { id: SLOT_SEPARATOR_ID }
        : { id: one.key, ...(one.isHidden ? { hidden: true } : {}) },
    ),
  }
}
