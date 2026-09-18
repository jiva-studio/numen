/**
 * The row moved: by a hand on it, by a wheel, and by being sent somewhere.
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
  readonly whereabouts: () => number
  /** The row sent to a place along itself, travelling or at once. */
  readonly send: (to: number, how: ScrollBehavior) => void
  /**
   * The row moved: how far along it is now, and whether it is standing still.
   * A row travelling to where it was sent passes over pages nobody turned to.
   */
  readonly isStill: () => boolean
  /** The hand on the row, and the wheel over it. */
  readonly onPointerDown: (event: PointerEvent) => void
  readonly onPointerMove: (event: PointerEvent) => void
  readonly letGo: (event: PointerEvent) => void
  readonly onWheel: (event: WheelEvent) => void
}

export function useHandScroll(area: Readonly<ShallowRef<HTMLElement | null>>): HandScroll {
  const along = ref(0)
  const isDragging = ref(false)

  /** Where the row was told to stand, while it is on its way there. */
  let heading: number | undefined

  const whereabouts = (): number => heading ?? area.value?.scrollLeft ?? along.value

  const send = (to: number, how: ScrollBehavior): void => {
    if (!area.value) return
    if (Math.abs(area.value.scrollLeft - to) <= THERE) return
    heading = to
    // A scroll that does not travel is over as soon as it is asked for.
    if (how === 'auto') along.value = to
    area.value.scrollTo({ left: to, behavior: how })
  }

  const isStill = (): boolean => {
    if (!area.value) return false
    along.value = area.value.scrollLeft
    if (heading === undefined) return true
    if (Math.abs(along.value - heading) > THERE) return false
    heading = undefined
    return true
  }

  /**
   * The row taken hold of and pulled. A book on a table is moved by putting a
   * hand on it, and a row five hundred pages long is a long way to travel by a
   * scrollbar.
   */
  const hand = new Hand()

  const onPointerDown = (event: PointerEvent): void => {
    // The controls sit over the room and are pressed, not dragged.
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
    // The hand has the row now, wherever it was being taken.
    heading = undefined
    area.value.scrollLeft = stood.x
    area.value.scrollTop = stood.y
    follow(event)
  }

  /**
   * The pointer followed where it leaves the room, so a hand that runs off the
   * edge still carries the row. A pointer the window is not holding is one this
   * cannot be asked about, and the drag then lasts as long as the pointer is
   * over the room.
   */
  const follow = (event: PointerEvent): void => {
    if (!area.value || area.value.hasPointerCapture(event.pointerId)) return
    try {
      area.value.setPointerCapture(event.pointerId)
    } catch {
      // The row is carried by the pointer while it is over the room.
    }
  }

  const letGo = (event: PointerEvent): void => {
    hand.release()
    isDragging.value = false
    if (area.value?.hasPointerCapture(event.pointerId)) {
      area.value.releasePointerCapture(event.pointerId)
    }
  }

  /**
   * A wheel turned. A row at rest has one axis and a wheel turned down means the
   * next page; drawn closer the room has both, and then down means down.
   */
  const onWheel = (event: WheelEvent): void => {
    if (!area.value) return
    const hasBelow = area.value.scrollHeight > area.value.clientHeight
    const by = getWheelOffset({ x: event.deltaX, y: event.deltaY }, hasBelow)
    if (by.x === 0 && by.y === 0) return
    event.preventDefault()
    // The wheel has the row now, wherever it was being taken.
    heading = undefined
    area.value.scrollLeft += by.x
    area.value.scrollTop += by.y
  }

  return {
    along,
    isDragging,
    whereabouts,
    send,
    isStill,
    onPointerDown,
    onPointerMove,
    letGo,
    onWheel,
  }
}
