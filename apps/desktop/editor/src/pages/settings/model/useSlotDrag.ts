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

const isOfGroup = (held: Pressed | null, group: SlotGroup): boolean =>
  held?.group.slot === group.slot

function measureSpans(list: Element | null | undefined): readonly RowSpan[] {
  return Array.from(list?.children ?? [], (one) => {
    const { top, height } = measureRect(one)
    return { top, height }
  })
}

/** What the drag of a row reads and writes: the rows of a slot and the place a row is moved to. */
export interface SlotDragSource {
  readonly getRows: (group: SlotGroup) => readonly SlotRow[]
  readonly move: (group: SlotGroup, from: number, to: number) => void
}

/**
 * The drag of a row of a slot. The held row follows the pointer by its own element, so a move of the
 * pointer renders nothing; the list renders when the place the row is over changes, and the rows
 * between slide aside. Rows move by their `top`, never by a transform or a layer of their own: a
 * composited row inside the clipped island is not painted in WebKitGTK while it moves.
 */
export function useSlotDrag(source: SlotDragSource) {
  const { settling, settle, finish } = useRowSettle()
  let pressed: Pressed | null = null
  let target = 0

  const clearHeldRow = (): void => {
    pressed?.row?.style.removeProperty('top')
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
      pressed.row.style.top = `${offset}px`
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

  const onGripPress = (group: SlotGroup, at: number, event: PointerEvent): void => {
    if (event.button !== 0) return
    const grip = event.currentTarget as HTMLElement | null
    pressed = {
      group,
      spans: measureSpans(grip?.closest('ul')),
      key: source.getRows(group)[at]?.key ?? '',
      startY: event.clientY,
      row: grip?.closest('li') ?? null,
    }
    target = at
    drag.startPress(at, event)
  }

  const onRowTransitionEnd = (event: TransitionEvent): void => {
    if (event.target === event.currentTarget && event.propertyName === 'top') finish()
  }

  const isPressed = (group: SlotGroup): boolean =>
    isOfGroup(pressed, group) && drag.dragging.value !== null

  const isDragging = (group: SlotGroup): boolean =>
    isOfGroup(pressed, group) && drag.dragging.value?.hasMoved === true && drag.at.value !== null

  const getRowClass = (group: SlotGroup, key: string): Record<string, boolean> => {
    const isHeld = pressed?.key === key
    const landing = settling.value?.key === key ? settling.value : null
    return {
      'plugin-slots__row--lifted': (isDragging(group) && isHeld) || landing?.isMoving === false,
      'plugin-slots__row--shifting': isDragging(group) && !isHeld,
      'plugin-slots__row--settling': landing?.isMoving === true,
    }
  }

  const getShift = (group: SlotGroup, key: string, at: number): number => {
    const from = drag.dragging.value?.item
    if (!pressed || !isOfGroup(pressed, group) || from === undefined || key === pressed.key)
      return 0
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
    return offset === 0 ? undefined : { top: `${offset}px` }
  }

  return { onGripPress, onRowTransitionEnd, isPressed, getRowClass, getRowStyle }
}
