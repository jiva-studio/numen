/**
 * A press that becomes a drag once the pointer has moved far enough.
 *
 * All event handling is here; the landing position and the meaning of a release are
 * the caller's.
 */
import { onScopeDispose, shallowRef, type ShallowRef } from 'vue'
import type { Position } from '@/shared/lib/math/geometry'
import type { Clock } from '@/shared/lib/time/clock'

/** What is being carried, and whether the pointer has gone far enough to mean it. */
interface Drag<Item> {
  readonly item: Item
  readonly hasMoved: boolean
}

/**
 * What following a press takes.
 *
 * `Item` is what the press picked up and `At` is where letting go would put it;
 * both are returned untouched.
 */
export interface Press<Item, At> {
  /** The distance the pointer moves before a press becomes a drag. */
  readonly getThreshold: () => number
  /** The clock the release is held against. */
  readonly getClock: () => Clock
  /** Where letting go here would put what is held. */
  readonly getDropTargetAt: (item: Item, point: Position) => At | null
  /**
   * What letting go after a drag evaluates to. The target is the one at the point of
   * release, in window coordinates, and nothing off any target.
   */
  readonly commit: (item: Item, target: At | null, point: Position) => void
  /** Called once, when the press becomes a drag. */
  readonly begin?: (item: Item) => void
  /** Called in place of `commit` when the system cancels a drag. */
  readonly cancel?: (item: Item) => void
}

/** What a press answers: what is held, where it would land, and how to start one. */
export interface PressDragState<Item, At> {
  readonly dragging: ShallowRef<Drag<Item> | null>
  readonly at: ShallowRef<At | null>
  /** Where the pointer is, for as long as a drag is live. */
  readonly position: ShallowRef<Position | null>
  readonly lift: (item: Item, event: PointerEvent) => void
}

/** The element that holds the pointer for the length of a press. */
interface Capture {
  readonly element: Element
  readonly pointer: number
}

export function usePressDrag<Item, At>(press: Press<Item, At>): PressDragState<Item, At> {
  const dragging = shallowRef<Drag<Item> | null>(null)
  const dropTarget = shallowRef<At | null>(null)
  const position = shallowRef<Position | null>(null)

  let start: Position
  let capture: Capture | null = null

  const detach = (): void => {
    window.removeEventListener('pointermove', drag)
    window.removeEventListener('pointerup', drop)
    window.removeEventListener('pointercancel', cancel)
    releaseCapture()
  }

  function drag(event: PointerEvent): void {
    const target = dragging.value!

    const now: Position = { x: event.clientX, y: event.clientY }
    const hasMoved =
      target.hasMoved ||
      Math.abs(now.x - start.x) > press.getThreshold() ||
      Math.abs(now.y - start.y) > press.getThreshold()

    dragging.value = { item: target.item, hasMoved }
    position.value = hasMoved ? now : null
    dropTarget.value = hasMoved ? press.getDropTargetAt(target.item, now) : null

    if (hasMoved && !target.hasMoved) press.begin?.(target.item)
  }

  function drop(event: PointerEvent): void {
    const target = dragging.value!
    const point: Position = { x: event.clientX, y: event.clientY }

    end()
    if (target.hasMoved) {
      press.commit(target.item, press.getDropTargetAt(target.item, point), point)
    }
  }

  function cancel(): void {
    const target = dragging.value!

    end()
    if (target.hasMoved) press.cancel?.(target.item)
  }

  function end(): void {
    detach()
    dropTarget.value = null
    position.value = null
    // Kept for one more frame: the click that follows the release reads it and is
    // suppressed.
    press.getClock().schedule(() => {
      dragging.value = null
    })
  }

  // WebKit keeps sending a captured pointer's events after it leaves the page; without
  // capture the release outside the window may never arrive.
  function takeCapture(event: PointerEvent): void {
    const element = event.currentTarget ?? event.target
    if (!(element instanceof Element)) return
    try {
      element.setPointerCapture(event.pointerId)
    } catch {
      // Thrown for a pointer that is not active, such as the pointer of a synthetic event.
      return
    }
    capture = { element, pointer: event.pointerId }
  }

  function releaseCapture(): void {
    if (!capture) return
    const { element, pointer } = capture
    capture = null
    if (element.hasPointerCapture(pointer)) element.releasePointerCapture(pointer)
  }

  const lift = (item: Item, event: PointerEvent): void => {
    start = { x: event.clientX, y: event.clientY }
    dragging.value = { item, hasMoved: false }
    takeCapture(event)
    window.addEventListener('pointermove', drag)
    window.addEventListener('pointerup', drop)
    window.addEventListener('pointercancel', cancel)
  }

  onScopeDispose(() => {
    detach()
    dropTarget.value = null
    position.value = null
    dragging.value = null
  })

  return { dragging, at: dropTarget, position, lift }
}
