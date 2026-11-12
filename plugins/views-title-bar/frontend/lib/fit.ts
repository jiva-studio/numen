import type { SlotItem } from '@numen/plugin-sdk'

/** One entry of a slot of the bar, by a key unique in the bar, and its width in pixels. */
export interface FitEntry {
  readonly key: string
  readonly kind: SlotItem['kind']
  readonly width: number
}

/** What the bar draws of its slots. */
interface BarFit {
  readonly hidden: ReadonlySet<string>
  readonly overflow: readonly string[]
}

const isMovable = (entry: FitEntry): boolean => entry.kind === 'command' || entry.kind === 'submenu'

function findLoneSeparators(row: readonly FitEntry[], hiddenKeys: ReadonlySet<string>): string[] {
  const lone: string[] = []
  let pending: FitEntry | null = null
  let hasBefore = false
  for (const entry of row) {
    if (entry.kind === 'separator') {
      if (pending || !hasBefore) lone.push(entry.key)
      else pending = entry
    } else if (!hiddenKeys.has(entry.key)) {
      hasBefore = true
      pending = null
    }
  }
  if (pending) lone.push(pending.key)
  return lone
}

function getHiddenEntries(rows: readonly (readonly FitEntry[])[], hiddenKeys: ReadonlySet<string>) {
  const all = new Set(hiddenKeys)
  for (const row of rows) for (const key of findLoneSeparators(row, hiddenKeys)) all.add(key)
  return all
}

function measureDrawn(
  rows: readonly (readonly FitEntry[])[],
  hiddenKeys: ReadonlySet<string>,
  moreWidth: number,
): number {
  const all = getHiddenEntries(rows, hiddenKeys)
  let width = 0
  let hasMore = false
  for (const entry of rows.flat()) {
    if (!all.has(entry.key)) width += entry.width
    else if (isMovable(entry)) hasMore = true
  }
  return hasMore ? width + moreWidth : width
}

/**
 * The entries of the slots of the bar that fit the room, the slots in the order drawn. When they do
 * not all fit, the commands and the submenus go into the menu from the end of the bar; once all are
 * there, the views are hidden from the end of the bar. A view never goes into the menu.
 */
export function fitBar(
  rows: readonly (readonly FitEntry[])[],
  room: number | null,
  moreWidth: number,
): BarFit {
  const removals = new Set<string>()
  if (room !== null) {
    const backwards = rows.flat().reverse()
    const queue = [
      ...backwards.filter(isMovable),
      ...backwards.filter((one) => one.kind === 'view'),
    ]
    for (const entry of queue) {
      if (measureDrawn(rows, removals, moreWidth) <= room) break
      removals.add(entry.key)
    }
  }
  return {
    hidden: getHiddenEntries(rows, removals),
    overflow: rows
      .flat()
      .filter((one) => isMovable(one) && removals.has(one.key))
      .map((one) => one.key),
  }
}
