import { browserClock, measureRect, usePressDrag, type Clock } from '@numen/ui'
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
  readonly keys: readonly string[]
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
  readonly move: (group: SlotGroup, from: number, to: number) => readonly SlotRow[]
}

/**
 * The drag of a row of a slot. The held row follows the pointer by its own element, so a move of the
 * pointer renders nothing; the list renders when the place the row is over changes, and the rows
 * between slide aside. The list is marked from the press, before the first move, so the rows are on
 * their own layers by the time one of them is lifted.
 */
export function useSlotDrag(source: SlotDragSource, clock: Clock = browserClock) {
  const { settling, settle, finish } = useRowSettle(clock)
  let pressed: Pressed | null = null
  let target = 0

  const isStale = (): boolean => {
    if (!pressed) return true
    const keys = source.getRows(pressed.group).map((one) => one.key)
    return keys.length !== pressed.keys.length || keys.some((one, at) => one !== pressed?.keys[at])
  }

  const clearHeldRow = (): void => {
    pressed?.row?.style.removeProperty('transform')
  }

  const drag = usePressDrag<number, number>({
    getThreshold: () => DRAG_THRESHOLD,
    getClock: () => clock,
    getDropTargetAt: (from, point) => {
      if (!pressed) return null
      target = getDropIndex(pressed.spans, from, point.y - pressed.startY, target) ?? from
      return target
    },
    isPositionTracked: () => false,
    move: (from, point) => {
      if (isStale()) {
        drag.abort()
        return
      }
      if (!pressed?.row) return
      const offset = clampRowOffset(pressed.spans, from, point.y - pressed.startY)
      pressed.row.style.transform = `translate3d(0, ${offset}px, 0)`
    },
    cancel: clearHeldRow,
    commit: (from, to, point) => {
      clearHeldRow()
      if (!pressed || to === null || isStale()) return
      const { group, spans, key, startY } = pressed
      const moved = source.move(group, from, to)
      const landed = moved.findIndex((one) => one.key === key)
      const offset =
        clampRowOffset(spans, from, point.y - startY) -
        getLandingShift(spans, from, landed < 0 ? to : landed)
      settle(key, offset)
    },
  })

  const onGripPress = (group: SlotGroup, at: number, event: PointerEvent): void => {
    if (event.button !== 0) return
    const grip = event.currentTarget as HTMLElement | null
    const keys = source.getRows(group).map((one) => one.key)
    pressed = {
      group,
      spans: measureSpans(grip?.closest('ul')),
      key: keys[at] ?? '',
      keys,
      startY: event.clientY,
      row: grip?.closest('li') ?? null,
    }
    target = at
    drag.startPress(at, event)
  }

  const onRowTransitionEnd = (event: TransitionEvent, key: string): void => {
    if (event.target === event.currentTarget && event.propertyName === 'transform') finish(key)
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
    return offset === 0 ? undefined : { transform: `translate3d(0, ${offset}px, 0)` }
  }

  return { onGripPress, onRowTransitionEnd, isPressed, getRowClass, getRowStyle }
}
