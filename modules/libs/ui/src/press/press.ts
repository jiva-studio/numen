/**
 * A press that becomes a drag once the pointer has travelled far enough.
 *
 * Everything that knows about events is here; where a landing is and what
 * letting go comes to are the caller's.
 */
import { onScopeDispose, shallowRef, type ShallowRef } from 'vue'
import type { Point } from '../lib/geometry'
import type { Environment } from '../lib/environment'

/** What is being carried, and whether the pointer has gone far enough to mean it. */
export interface Pressing<Held> {
  readonly held: Held
  readonly moved: boolean
}

/**
 * What following a press takes.
 *
 * `Held` is what the press picked up and `At` is where letting go would put it;
 * both are handed back untouched.
 */
export interface Press<Held, At> {
  /** How far the pointer travels before a press becomes a drag. */
  readonly threshold: () => number
  /** The clock the release is held against. */
  readonly environment: () => Environment
  /** Where letting go at this point would put what is held. */
  readonly landingAt: (held: Held, at: Point) => At | null
  /** What letting go after a drag comes to. The landing is nothing off any target. */
  readonly settle: (held: Held, at: At | null) => void
  /** Said once, when the press turns into a drag. */
  readonly began?: (held: Held) => void
}

/** What a press answers: what is held, where it would land, and how to start one. */
export interface Pressed<Held, At> {
  readonly dragging: ShallowRef<Pressing<Held> | null>
  readonly at: ShallowRef<At | null>
  /** Where the pointer is, for as long as a drag is live. */
  readonly point: ShallowRef<Point | null>
  readonly lift: (held: Held, event: PointerEvent) => void
}

export function usePressDrag<Held, At>(press: Press<Held, At>): Pressed<Held, At> {
  const dragging = shallowRef<Pressing<Held> | null>(null)
  const at = shallowRef<At | null>(null)
  const point = shallowRef<Point | null>(null)

  let start: Point = { x: 0, y: 0 }

  const detach = (): void => {
    window.removeEventListener('pointermove', drag)
    window.removeEventListener('pointerup', drop)
    window.removeEventListener('pointercancel', drop)
  }

  function drag(event: PointerEvent): void {
    const held = dragging.value
    if (!held) return

    const now: Point = { x: event.clientX, y: event.clientY }
    const moved =
      held.moved ||
      Math.abs(now.x - start.x) > press.threshold() ||
      Math.abs(now.y - start.y) > press.threshold()

    dragging.value = { held: held.held, moved }
    point.value = moved ? now : null
    at.value = moved ? press.landingAt(held.held, now) : null

    if (moved && !held.moved) press.began?.(held.held)
  }

  function drop(): void {
    const held = dragging.value
    const found = at.value

    detach()
    at.value = null
    point.value = null
    if (held?.moved) press.settle(held.held, found)
    // Held one frame longer: the click that follows the release reads it and
    // stands down.
    press.environment().schedule(() => {
      dragging.value = null
    })
  }

  const lift = (held: Held, event: PointerEvent): void => {
    start = { x: event.clientX, y: event.clientY }
    dragging.value = { held, moved: false }
    window.addEventListener('pointermove', drag)
    window.addEventListener('pointerup', drop)
    window.addEventListener('pointercancel', drop)
  }

  onScopeDispose(() => {
    detach()
    at.value = null
    point.value = null
    dragging.value = null
  })

  return { dragging, at, point, lift }
}
