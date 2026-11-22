import { getRowShifts, type RowSpan } from '../lib/slotDrag'
import type { SlotRow } from '../lib/slotLayout'
import type { SlotGroup } from '../types'

/** The row held by the pointer, as the look of the rows reads it. */
export interface HeldRow {
  readonly group: SlotGroup
  readonly spans: readonly RowSpan[]
  readonly key: string
}

/** What the look of the rows reads: the held row, the place it is over and the row just dropped. */
export interface RowLookSource {
  readonly getHeld: () => HeldRow | null
  readonly getFrom: () => number | undefined
  readonly getOver: () => number | null
  readonly isMoved: () => boolean
  readonly getSettling: () => {
    readonly key: string
    readonly offset: number
    readonly isMoving: boolean
  } | null
  readonly getRows: (group: SlotGroup) => readonly SlotRow[]
}

const isOfGroup = (held: HeldRow | null, group: SlotGroup): held is HeldRow =>
  held?.group.slot === group.slot

/** The classes and the transform of each row of a slot while a row is held, shifting aside or settling. */
export function createRowLook(source: RowLookSource) {
  const isDragging = (group: SlotGroup): boolean =>
    isOfGroup(source.getHeld(), group) && source.isMoved() && source.getOver() !== null

  const getRowClass = (group: SlotGroup, key: string): Record<string, boolean> => {
    const isHeld = source.getHeld()?.key === key
    const settling = source.getSettling()
    const landing = settling?.key === key ? settling : null
    return {
      'plugin-slots__row--lifted': (isDragging(group) && isHeld) || landing?.isMoving === false,
      'plugin-slots__row--shifting': isDragging(group) && !isHeld,
      'plugin-slots__row--settling': landing?.isMoving === true,
    }
  }

  const getShift = (group: SlotGroup, key: string, at: number): number => {
    const held = source.getHeld()
    const from = source.getFrom()
    if (!isOfGroup(held, group) || from === undefined || key === held.key) return 0
    const pitch = held.spans[from]?.height ?? 0
    return (
      getRowShifts(source.getRows(group).length, from, source.getOver() ?? from, pitch)[at] ?? 0
    )
  }

  const getRowStyle = (
    group: SlotGroup,
    key: string,
    at: number,
  ): Record<string, string> | undefined => {
    const landing = source.getSettling()
    const offset =
      landing?.key === key ? (landing.isMoving ? 0 : landing.offset) : getShift(group, key, at)
    return offset === 0 ? undefined : { transform: `translate3d(0, ${offset}px, 0)` }
  }

  return { isDragging, getRowClass, getRowStyle }
}
