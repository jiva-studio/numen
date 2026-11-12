/**
 * Scroll position changes of the page strip: by pointer drag, by wheel, and by
 * programmatic scrolling.
 *
 * The wheel offset (`getWheelOffset`) and the drag-to-scroll offset are computed
 * in the lib layer; this module applies those results to the scroll element.
 */
import { ref, type Ref, type ShallowRef } from 'vue'
import { Hand, getWheelOffset } from '../lib/hand'

/** How near the row has to be to count as standing where it was sent, in CSS pixels. */
const THERE = 1

/** What the strip is doing, how to move it, and the pointer and wheel handlers that move it. */
export interface PanScroll {
  readonly state: {
    readonly offset: Ref<number>
    readonly isDragging: Ref<boolean>
  }
  readonly moves: {
    readonly getScrollOffset: () => number
    readonly send: (to: number, how: ScrollBehavior) => void
    readonly restore: () => void
    readonly isMoved: () => boolean
    readonly isStill: () => boolean
  }
  readonly drag: {
    readonly onPointerDown: (event: PointerEvent) => void
    readonly onPointerMove: (event: PointerEvent) => void
    readonly letGo: (event: PointerEvent) => void
  }
  readonly onWheel: (event: WheelEvent) => void
}

function followPointer(element: HTMLElement | null, event: PointerEvent): boolean {
  if (!element) return false
  if (element.hasPointerCapture(event.pointerId)) return true
  try {
    element.setPointerCapture(event.pointerId)
    return true
  } catch {
    return false
  }
}

function applyWheelScroll(area: HTMLElement, event: WheelEvent): boolean {
  const hasBelow = area.scrollHeight > area.clientHeight
  const delta = getWheelOffset({ x: event.deltaX, y: event.deltaY }, hasBelow)
  if (delta.x === 0 && delta.y === 0) return false
  event.preventDefault()
  area.scrollLeft += delta.x
  area.scrollTop += delta.y
  return true
}

function createPointerDrag(
  area: Readonly<ShallowRef<HTMLElement | null>>,
  isDragging: Ref<boolean>,
  onInterrupted: () => void,
) {
  const hand = new Hand()

  const onPointerDown = (event: PointerEvent): void => {
    if (!area.value || event.button !== 0) return
    hand.take(
      { x: event.clientX, y: event.clientY },
      { x: area.value.scrollLeft, y: area.value.scrollTop },
    )
  }

  const onPointerMove = (event: PointerEvent): void => {
    if (!area.value || !hand.holding) return
    const status = hand.to({ x: event.clientX, y: event.clientY })
    if (!status) return
    isDragging.value = true
    onInterrupted()
    area.value.scrollLeft = status.x
    area.value.scrollTop = status.y
    followPointer(area.value, event)
  }

  const letGo = (event: PointerEvent): void => {
    hand.release()
    isDragging.value = false
    if (area.value?.hasPointerCapture(event.pointerId)) {
      area.value.releasePointerCapture(event.pointerId)
    }
  }

  return { onPointerDown, onPointerMove, letGo }
}

export function usePanScroll(area: Readonly<ShallowRef<HTMLElement | null>>): PanScroll {
  const axisOffset = ref(0)
  const isDragging = ref(false)

  /** The programmatic scroll target while a scroll towards it is in progress. */
  let heading: number | undefined

  /** Whether `heading` was sent as a travel, which the browser ends when the row is moved. */
  let isTravel = false

  const getScrollOffset = (): number => heading ?? area.value?.scrollLeft ?? axisOffset.value

  const send = (to: number, how: ScrollBehavior): void => {
    if (!area.value) return
    if (Math.abs(area.value.scrollLeft - to) <= THERE) {
      // The row is at the place already. An instant scroll there ends a travel still in flight to
      // another place, and no target is left that no scroll event would clear.
      heading = undefined
      area.value.scrollTo({ left: to, behavior: 'auto' })
      return
    }
    heading = to
    isTravel = how === 'smooth'
    // An instant scroll completes immediately.
    if (how === 'auto') axisOffset.value = to
    area.value.scrollTo({ left: to, behavior: how })
  }

  const restore = (): void => {
    if (!area.value || (heading !== undefined && !isTravel)) return
    const goal = heading ?? axisOffset.value
    heading = undefined
    if (Math.abs(area.value.scrollLeft - goal) <= THERE) return
    axisOffset.value = goal
    area.value.scrollLeft = goal
  }

  const isMoved = (): boolean => {
    if (!area.value) return false
    return Math.abs(area.value.scrollLeft - (heading ?? axisOffset.value)) > THERE
  }

  const isStill = (): boolean => {
    if (!area.value) return false
    axisOffset.value = area.value.scrollLeft
    if (Math.abs(axisOffset.value - (heading ?? axisOffset.value)) > THERE) return false
    heading = undefined
    return true
  }

  const drag = createPointerDrag(area, isDragging, () => {
    heading = undefined
  })

  const onWheel = (event: WheelEvent): void => {
    if (!area.value) return
    if (applyWheelScroll(area.value, event)) heading = undefined
  }

  const state: PanScroll = {
    state: { offset: axisOffset, isDragging },
    moves: { getScrollOffset, send, restore, isMoved, isStill },
    drag,
    onWheel,
  }
  return state
}
