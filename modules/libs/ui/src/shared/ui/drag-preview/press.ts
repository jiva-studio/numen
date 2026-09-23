/**
 * A press that becomes a drag once the pointer has travelled far enough.
 *
 * Everything that knows about events is here; where a landing is and what
 * letting go comes to are the caller's.
 */
import { onScopeDispose, shallowRef, type ShallowRef } from 'vue'
import type { Position } from '@/shared/lib/geometry'
import type { Clock } from '@/shared/lib/clock'

/** What is being carried, and whether the pointer has gone far enough to mean it. */
export interface Drag<Item> {
  readonly item: Item
  readonly hasMoved: boolean
}

/**
 * What following a press takes.
 *
 * `Item` is what the press picked up and `At` is where letting go would put it;
 * both are handed back untouched.
 */
export interface Press<Item, At> {
  /** How far the pointer travels before a press becomes a drag. */
  readonly getThreshold: () => number
  /** The clock the release is held against. */
  readonly getClock: () => Clock
  /** Where letting go here would put what is held. */
  readonly getLandingAt: (item: Item, at: Position) => At | null
  /** What letting go after a drag comes to. The landing is nothing off any target. */
  readonly settle: (item: Item, at: At | null) => void
  /** Said once, when the press turns into a drag. */
  readonly begin?: (item: Item) => void
}

/** What a press answers: what is held, where it would land, and how to start one. */
export interface PressDragState<Item, At> {
  readonly dragging: ShallowRef<Drag<Item> | null>
  readonly at: ShallowRef<At | null>
  /** Where the pointer is, for as long as a drag is live. */
  readonly position: ShallowRef<Position | null>
  readonly lift: (item: Item, event: PointerEvent) => void
}

export function usePressDrag<Item, At>(press: Press<Item, At>): PressDragState<Item, At> {
  const dragging = shallowRef<Drag<Item> | null>(null)
  const at = shallowRef<At | null>(null)
  const position = shallowRef<Position | null>(null)

  let start: Position = { x: 0, y: 0 }

  const detach = (): void => {
    window.removeEventListener('pointermove', drag)
    window.removeEventListener('pointerup', drop)
    window.removeEventListener('pointercancel', drop)
  }

  function drag(event: PointerEvent): void {
    const held = dragging.value
    if (!held) return

    const now: Position = { x: event.clientX, y: event.clientY }
    const hasMoved =
      held.hasMoved ||
      Math.abs(now.x - start.x) > press.getThreshold() ||
      Math.abs(now.y - start.y) > press.getThreshold()

    dragging.value = { item: held.item, hasMoved }
    position.value = hasMoved ? now : null
    at.value = hasMoved ? press.getLandingAt(held.item, now) : null

    if (hasMoved && !held.hasMoved) press.begin?.(held.item)
  }

  function drop(): void {
    const held = dragging.value
    const found = at.value

    detach()
    at.value = null
    position.value = null
    if (held?.hasMoved) press.settle(held.item, found)
    // Held one frame longer: the click that follows the release reads it and
    // stands down.
    press.getClock().schedule(() => {
      dragging.value = null
    })
  }

  const lift = (item: Item, event: PointerEvent): void => {
    start = { x: event.clientX, y: event.clientY }
    dragging.value = { item, hasMoved: false }
    window.addEventListener('pointermove', drag)
    window.addEventListener('pointerup', drop)
    window.addEventListener('pointercancel', drop)
  }

  onScopeDispose(() => {
    detach()
    at.value = null
    position.value = null
    dragging.value = null
  })

  return { dragging, at, position, lift }
}
