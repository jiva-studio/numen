import { browserClock, measureRect, usePressDrag } from '@numen/ui'
import {
  clampRowOffset,
  getDropIndex,
  getLandingShift,
  getRowShifts,
  type RowSpan,
} from '../lib/slotDrag'
import type { SlotRow } from '../lib/slotLayout'
import type { SlotGroup } from '../types'
import { useRowSettle } from './useRowSettle'

const DRAG_THRESHOLD = 4

interface Pressed {
  readonly group: SlotGroup
  readonly spans: readonly RowSpan[]
  readonly key: string
  readonly startY: number
  readonly row: HTMLElement | null
}

/** What the drag of a row reads and writes: the rows of a slot and the place a row is moved to. */
export interface SlotDragSource {
  readonly getRows: (group: SlotGroup) => readonly SlotRow[]
  readonly move: (group: SlotGroup, from: number, to: number) => void
}

/**
 * The drag of a row of a slot. The held row follows the pointer by its own element, so a move of the
 * pointer renders nothing; the list renders when the place the row is over changes, and the rows
 * between slide aside.
 */
export function useSlotDrag(source: SlotDragSource) {
  const { settling, settle, finish } = useRowSettle()
  let pressed: Pressed | null = null
  let target = 0

  const clearHeldRow = (): void => {
    pressed?.row?.style.removeProperty('transform')
  }

  const drag = usePressDrag<number, number>({
    getThreshold: () => DRAG_THRESHOLD,
    getClock: () => browserClock,
    getDropTargetAt: (from, point) => {
      if (!pressed) return null
      target = getDropIndex(pressed.spans, from, point.y - pressed.startY, target) ?? from
      return target
    },
    isPositionTracked: () => false,
    move: (from, point) => {
      if (!pressed?.row) return
      const offset = clampRowOffset(pressed.spans, from, point.y - pressed.startY)
      pressed.row.style.transform = `translateY(${offset}px)`
    },
    cancel: clearHeldRow,
    commit: (from, to, point) => {
      clearHeldRow()
      if (!pressed || to === null) return
      const { group, spans, key, startY } = pressed
      const offset =
        clampRowOffset(spans, from, point.y - startY) - getLandingShift(spans, from, to)
      settle(key, offset)
      source.move(group, from, to)
    },
  })

  const hasMoved = (): boolean => drag.dragging.value?.hasMoved === true

  const onGripPress = (group: SlotGroup, at: number, event: PointerEvent): void => {
    if (event.button !== 0) return
    const grip = event.currentTarget as HTMLElement | null
    const spans = Array.from(grip?.closest('ul')?.children ?? [], (one) => {
      const { top, height } = measureRect(one)
      return { top, height }
    })
    pressed = {
      group,
      spans,
      key: source.getRows(group)[at]?.key ?? '',
      startY: event.clientY,
      row: grip?.closest('li') ?? null,
    }
    target = at
    drag.startPress(at, event)
  }

  const onRowTransitionEnd = (event: TransitionEvent): void => {
    if (event.target === event.currentTarget && event.propertyName === 'transform') finish()
  }

  const isHeld = (group: SlotGroup, key: string): boolean =>
    pressed?.group === group && pressed.key === key && hasMoved()

  const isShifting = (group: SlotGroup): boolean => pressed?.group === group && hasMoved()

  const getShift = (group: SlotGroup, key: string, at: number): number => {
    const from = drag.dragging.value?.item
    if (!pressed || pressed.group !== group || from === undefined || key === pressed.key) return 0
    const pitch = pressed.spans[from]?.height ?? 0
    return getRowShifts(source.getRows(group).length, from, drag.at.value ?? from, pitch)[at] ?? 0
  }

  const getRowStyle = (
    group: SlotGroup,
    key: string,
    at: number,
  ): Record<string, string> | undefined => {
    const landing = settling.value
    const offset =
      landing?.key === key ? (landing.isMoving ? 0 : landing.offset) : getShift(group, key, at)
    return offset === 0 ? undefined : { transform: `translateY(${offset}px)` }
  }

  return { settling, onGripPress, onRowTransitionEnd, isHeld, isShifting, getRowStyle }
}
