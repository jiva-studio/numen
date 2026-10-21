/**
 * The row hasMoved: by a hand on it, by a wheel, and by being sent somewhere.
 *
 * Which way a wheel moves the row and where a hand leaves it are `hand.ts`;
 * this is the element those answers are applied to.
 */
import { ref, type Ref, type ShallowRef } from 'vue'
import { Hand, getWheelOffset } from '../lib/hand'

/** How near the row has to be to count as standing where it was sent, in CSS pixels. */
const THERE = 1

export interface HandScroll {
  /** How far the row has been scrolled, in CSS pixels. */
  readonly along: Ref<number>
  /** Whether the hand is dragging, which is what the room is drawn as. */
  readonly isDragging: Ref<boolean>
  /**
   * How far along the row is, or will be: where it was sent, and otherwise
   * where the room says it stands. A scroll moves the room before an event
   * reports it.
   */
  readonly getScrollOffset: () => number
  /** The row sent to a place along itself, travelling or at once. */
  readonly send: (to: number, how: ScrollBehavior) => void
  /**
   * The row hasMoved: how far along it is now, and whether it is standing still.
   * A row travelling to where it was sent passes over pages nobody turned to.
   */
  readonly isStill: () => boolean
  /** The hand on the row, and the wheel over it. */
  readonly onPointerDown: (event: PointerEvent) => void
  readonly onPointerMove: (event: PointerEvent) => void
  readonly letGo: (event: PointerEvent) => void
  readonly onWheel: (event: WheelEvent) => void
}

function followPointer(element: HTMLElement | null, event: PointerEvent): void {
  if (!element || element.hasPointerCapture(event.pointerId)) return
  try {
    element.setPointerCapture(event.pointerId)
  } catch {
    // The row is carried by the pointer while it is over the room.
  }
}

function applyWheelScroll(area: HTMLElement, event: WheelEvent): boolean {
  const hasBelow = area.scrollHeight > area.clientHeight
  const by = getWheelOffset({ x: event.deltaX, y: event.deltaY }, hasBelow)
  if (by.x === 0 && by.y === 0) return false
  event.preventDefault()
  area.scrollLeft += by.x
  area.scrollTop += by.y
  return true
}

function usePointerDrag(
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
    const stood = hand.to({ x: event.clientX, y: event.clientY })
    if (!stood) return
    isDragging.value = true
    onInterrupted()
    area.value.scrollLeft = stood.x
    area.value.scrollTop = stood.y
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

export function useHandScroll(area: Readonly<ShallowRef<HTMLElement | null>>): HandScroll {
  const alongOffset = ref(0)
  const isDragging = ref(false)

  /** Where the row was told to stand, while it is on its way there. */
  let heading: number | undefined

  const getScrollOffset = (): number => heading ?? area.value?.scrollLeft ?? alongOffset.value

  const send = (to: number, how: ScrollBehavior): void => {
    if (!area.value) return
    if (Math.abs(area.value.scrollLeft - to) <= THERE) return
    heading = to
    // A scroll that does not travel is over as soon as it is asked for.
    if (how === 'auto') alongOffset.value = to
    area.value.scrollTo({ left: to, behavior: how })
  }

  const isStill = (): boolean => {
    if (!area.value) return false
    along.value = area.value.scrollLeft
    if (Math.abs(along.value - (heading ?? along.value)) > THERE) return false
    heading = undefined
    return true
  }

  const { onPointerDown, onPointerMove, letGo } = usePointerDrag(area, isDragging, () => {
    heading = undefined
  })

  const onWheel = (event: WheelEvent): void => {
    if (!area.value) return
    if (applyWheelScroll(area.value, event)) heading = undefined
  }

  return {
    along,
    isDragging,
    getScrollOffset,
    send,
    isStill,
    onPointerDown,
    onPointerMove,
    letGo,
    onWheel,
  }
}
