/**
 * The pointer. Everything that knows about events and screen pixels is here;
 * what a gesture *means* is worked out in `arrange/drop.ts`, as a value.
 */
import { computed, onScopeDispose, ref, type ComputedRef, type Ref } from 'vue'
import { resolveDrop, getRelationWithoutDirection, type Drop } from '../../lib/arrange'
import type { PlexOptions } from '../../lib/arrange'
import type { PlexFrame } from '../../lib/frame'
import type { Position } from '../../lib/node'
import type { PlexRelation } from '../../lib/relation'

export interface Gesture {
  /** The node it started from, while one is under way. */
  readonly from: Ref<string | null>
  /** Where the pointer is, in the plex's own coordinates. */
  readonly at: Ref<Position | null>
  /** What letting go here would come to, so it can be shown before it does. */
  readonly outcome: Ref<Drop | null>
}

export interface GestureHandlers {
  readonly begin: (from: string, event: PointerEvent) => void
  /**
   * Reach out from a node with nowhere to point — from the keyboard, where
   * there is no direction to read. It evaluates to what a press that never
   * travelled evaluates to, and is over as soon as it is asked for.
   */
  readonly ask: (from: string) => void
  readonly cancel: () => void
}

/**
 * Screen to plex coordinates.
 *
 * Through the element's own matrix, so the scale it is drawn at is whatever
 * the element says it is.
 */
export function getPosition(svg: SVGSVGElement, event: PointerEvent): Position | null {
  const screen = svg.getScreenCTM?.()
  if (!screen) return null
  const m = screen.inverse()
  return {
    x: m.a * event.clientX + m.c * event.clientY + m.e,
    y: m.b * event.clientX + m.d * event.clientY + m.f,
  }
}

export interface PlexGestureDeps {
  readonly frame: () => PlexFrame
  readonly options: () => PlexOptions
  readonly relations: () => readonly PlexRelation[]
  readonly threshold: () => number
  readonly commit: (drop: Drop) => void
}

interface GesturePoints {
  readonly from: Ref<string | null>
  readonly at: Ref<Position | null>
  readonly start: Ref<Position | null>
}

interface GestureListenerHandlers {
  readonly onMove: (event: PointerEvent) => void
  readonly onFinish: () => void
  readonly onCancel: () => void
}

function releasePointer(element: SVGSVGElement | null, pointerId: number | null): void {
  if (element && pointerId !== null) {
    try {
      // Stryker disable next-line OptionalChaining: a missing method throws a TypeError, which the catch below absorbs
      element.releasePointerCapture?.(pointerId)
    } catch {
      // Never held it; nothing to release.
    }
  }
}

function capturePointer(element: SVGSVGElement, pointerId: number): void {
  try {
    // Stryker disable next-line OptionalChaining: a missing method throws a TypeError, which the catch below absorbs
    element.setPointerCapture?.(pointerId)
  } catch {
    // Nothing to hold on to; the listeners below still follow it.
  }
}

function listenGesture(pointerId: number, handlers: GestureListenerHandlers): () => void {
  const isMine = (event: PointerEvent) => event.pointerId === pointerId

  const onMove = (event: PointerEvent) => {
    if (isMine(event)) handlers.onMove(event)
  }
  const onUp = (up: PointerEvent) => {
    if (isMine(up)) handlers.onFinish()
  }
  const onLost = (event: PointerEvent) => {
    if (isMine(event)) handlers.onCancel()
  }
  const onKey = (key: KeyboardEvent) => {
    if (key.key === 'Escape') handlers.onCancel()
  }

  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onLost)
  window.addEventListener('keydown', onKey)

  return () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onLost)
    window.removeEventListener('keydown', onKey)
  }
}

function useGestureOutcome(points: GesturePoints, deps: PlexGestureDeps): ComputedRef<Drop | null> {
  return computed(() => {
    const source = points.from.value
    const now = points.at.value
    if (!source || !now) return null

    const began = points.start.value
    const travelled = !!began && Math.hypot(now.x - began.x, now.y - began.y) >= deps.threshold()
    if (!travelled) {
      const relation = getRelationWithoutDirection(deps.relations())
      return relation ? { kind: 'create', from: source, relation } : null
    }

    return resolveDrop({
      frame: deps.frame(),
      options: deps.options(),
      from: source,
      point: now,
      relations: deps.relations(),
    })
  })
}

/**
 * Follow a drag from a node until it is let go or given up on.
 *
 * A click without travel is a gesture too: it is the commonest thing anyone
 * wants — one more child — and asking for a drag to get it would make the
 * handle useless to a trackpad and to anyone who cannot hold a button down.
 */
export function usePlexGesture(deps: PlexGestureDeps): Gesture & GestureHandlers {
  const surface = ref<SVGSVGElement | null>(null)
  const from = ref<string | null>(null)
  const pointerPosition = ref<Position | null>(null)
  const start = ref<Position | null>(null)
  const pointer = ref<number | null>(null)
  let detach: (() => void) | null = null

  const outcome = useGestureOutcome({ from, at: pointerPosition, start }, deps)

  const stop = () => {
    detach?.()
    detach = null
    releasePointer(surface.value, pointer.value)
    from.value = null
    pointerPosition.value = null
    start.value = null
    pointer.value = null
  }

  const begin = (source: string, event: PointerEvent) => {
    const element = (event.target as SVGElement | null)?.ownerSVGElement ?? null
    const began = element ? getPosition(element, event) : null
    if (!element || !began) return

    surface.value = element
    from.value = source
    start.value = began
    pointerPosition.value = began
    pointer.value = event.pointerId
    capturePointer(element, event.pointerId)

    detach = listenGesture(event.pointerId, {
      onMove: (e) => {
        // Stryker disable next-line ConditionalExpression: the surface is set before any listener that calls this is installed
        if (surface.value) pointerPosition.value = getPosition(surface.value, e)
      },
      onFinish: () => {
        const resolved = outcome.value
        stop()
        if (resolved) deps.commit(resolved)
      },
      onCancel: stop,
    })
  }

  const ask = (source: string) => {
    const relation = getRelationWithoutDirection(deps.relations())
    if (relation) deps.commit({ kind: 'create', from: source, relation })
  }

  onScopeDispose(stop)

  return { from, at: pointerPosition, outcome, begin, ask, cancel: stop }
}
