/**
 * Scroll position changes of the page strip: by pointer drag, by wheel, and by
 * programmatic scrolling.
 *
 * The scroll direction of a wheel and the drag-to-scroll offset are computed in
 * `hand.ts`; this module applies those results to the scroll element.
 */
import { ref, type Ref, type ShallowRef } from 'vue'
import { Hand, getWheelOffset } from '../lib/hand'

/** How near the row has to be to count as standing where it was sent, in CSS pixels. */
const THERE = 1

export interface PanScroll {
  /** How far the row has been scrolled, in CSS pixels. */
  readonly along: Ref<number>
  /** Whether a pointer drag is in progress, which determines the cursor style. */
  readonly isDragging: Ref<boolean>
  /**
   * The scroll offset of the strip, or the target offset when a programmatic
   * scroll is in progress. A scroll updates the element before an event
   * reports it.
   */
  readonly getScrollOffset: () => number
  /** The row sent to a place along itself, travelling or at once. */
  readonly send: (to: number, how: ScrollBehavior) => void
  /**
   * Whether the strip is stationary. A strip scrolling towards a target passes
   * pages the user did not navigate to.
   */
  readonly isStill: () => boolean
  /** Pointer drag and wheel handlers for the strip. */
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
    // The pointer stays captured by the strip while it is over the scroll area.
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

  const getScrollOffset = (): number => heading ?? area.value?.scrollLeft ?? axisOffset.value

  const send = (to: number, how: ScrollBehavior): void => {
    if (!area.value) return
    if (Math.abs(area.value.scrollLeft - to) <= THERE) return
    heading = to
    // An instant scroll completes immediately.
    if (how === 'auto') axisOffset.value = to
    area.value.scrollTo({ left: to, behavior: how })
  }

  const isStill = (): boolean => {
    if (!area.value) return false
    axisOffset.value = area.value.scrollLeft
    if (Math.abs(axisOffset.value - (heading ?? axisOffset.value)) > THERE) return false
    heading = undefined
    return true
  }

  const { onPointerDown, onPointerMove, letGo } = createPointerDrag(area, isDragging, () => {
    heading = undefined
  })

  const onWheel = (event: WheelEvent): void => {
    if (!area.value) return
    if (applyWheelScroll(area.value, event)) heading = undefined
  }

  const state: PanScroll = {
    along: axisOffset,
    isDragging,
    getScrollOffset,
    send,
    isStill,
    onPointerDown,
    onPointerMove,
    letGo,
    onWheel,
  }
  return state
}
