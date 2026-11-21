import type { JsonValue } from '@/shared/api/json'
import type { SlotGroup } from '../types'

/** One entry of a slot as the settings page draws it, in the order the slot draws it. */
export interface SlotRow {
  readonly key: string
  readonly label: string
  readonly pluginName: string
  readonly isHidden: boolean
}

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

/** The entries of a slot: those the layout names in its order, then the others in the order the slot draws them. */
export function getSlotRows(group: SlotGroup, layout: JsonValue | undefined): readonly SlotRow[] {
  const stored = getStoredEntries(layout, group.slot)
  const position = new Map(stored.map((one, at) => [one.id, at]))
  const hidden = new Set(stored.filter((one) => one.hidden === true).map((one) => one.id))
  const rank = (key: string): number => position.get(key) ?? stored.length
  return group.contributors
    .map((one, at) => ({ one, at }))
    .sort((a, b) => rank(a.one.key) - rank(b.one.key) || a.at - b.at)
    .map(({ one }) => ({ ...one, isHidden: hidden.has(one.key) }))
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

/** The rows with the visibility of the one at `at` turned over. */
export function toggleSlotRow(rows: readonly SlotRow[], at: number): readonly SlotRow[] {
  return rows.map((one, index) => (index === at ? { ...one, isHidden: !one.isHidden } : one))
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
    [slot]: rows.map((one) => ({ id: one.key, ...(one.isHidden ? { hidden: true } : {}) })),
  }
}
