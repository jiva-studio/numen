/**
 * The pointer. Everything that knows about events and screen pixels is here;
 * what a gesture *means* is worked out in `arrange/drop.ts`, as a value.
 */
import { computed, onScopeDispose, ref, type Ref } from 'vue'
import { resolveDrop, seatWithoutDirection, type Drop } from '../lib/arrange'
import type { PlexOptions } from '../lib/arrange'
import type { PlexFrame } from '../lib/frame'
import type { Position } from '../lib/node'
import type { PlexRelatedSeat } from '../lib/seat'

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
   * there is no direction to read. It comes to what a press that never
   * travelled comes to, and is over as soon as it is asked for.
   */
  readonly ask: (from: string) => void
  readonly cancel: () => void
}

/**
 * Screen to plex coordinates.
 *
 * Through the element's own matrix rather than by measuring the box: the plex
 * happens to be drawn at one unit to the pixel today, and arithmetic that
 * assumed so would break silently the first time that changed.
 */
export function positionIn(svg: SVGSVGElement, event: PointerEvent): Position | null {
  const screen = svg.getScreenCTM?.()
  if (!screen) return null
  const m = screen.inverse()
  return {
    x: m.a * event.clientX + m.c * event.clientY + m.e,
    y: m.b * event.clientX + m.d * event.clientY + m.f,
  }
}

/**
 * Follow a drag from a node until it is let go or given up on.
 *
 * A click without travel is a gesture too: it is the commonest thing anyone
 * wants — one more child — and asking for a drag to get it would make the
 * handle useless to a trackpad and to anyone who cannot hold a button down.
 */
export function usePlexGesture(
  frame: () => PlexFrame,
  options: () => PlexOptions,
  getSeats: () => readonly PlexRelatedSeat[],
  threshold: () => number,
  settle: (drop: Drop) => void,
): Gesture & GestureHandlers {
  const surface = ref<SVGSVGElement | null>(null)
  const svg = () => surface.value

  const from = ref<string | null>(null)
  const at = ref<Position | null>(null)
  const start = ref<Position | null>(null)
  const pointer = ref<number | null>(null)

  /** Far enough from where it started to be a drag rather than a click. */
  const hasTravelled = (now: Position) => {
    const began = start.value
    return !!began && Math.hypot(now.x - began.x, now.y - began.y) >= threshold()
  }

  const outcome = computed<Drop | null>(() => {
    const source = from.value
    const now = at.value
    if (!source || !now) return null

    // A gesture that has not travelled is a press, and a press has a rule
    // rather than a direction.
    if (!hasTravelled(now)) {
      const seat = seatWithoutDirection(getSeats())
      return seat ? { kind: 'create', from: source, seat } : null
    }

    return resolveDrop({
      frame: frame(),
      options: options(),
      from: source,
      at: now,
      seats: getSeats(),
    })
  })

  /** What the gesture under way installed on the window, if anything. */
  let detach: (() => void) | null = null

  const stop = (element: SVGSVGElement | null) => {
    // Every way a gesture ends comes through here, so this is the one place
    // that has to let go of everything: a listener left behind goes on
    // answering for a gesture nobody is making.
    detach?.()
    detach = null
    if (element && pointer.value !== null) {
      try {
        element.releasePointerCapture?.(pointer.value)
      } catch {
        // Never held it; nothing to let go of.
      }
    }
    from.value = null
    at.value = null
    start.value = null
    pointer.value = null
  }

  const move = (event: PointerEvent) => {
    const element = svg()
    if (!element) return
    at.value = positionIn(element, event)
  }

  const finish = () => {
    const settled = outcome.value
    stop(svg())
    if (settled) settle(settled)
  }

  const cancel = () => stop(svg())

  const ask = (source: string) => {
    const seat = seatWithoutDirection(getSeats())
    if (seat) settle({ kind: 'create', from: source, seat })
  }

  const begin = (source: string, event: PointerEvent) => {
    // The element comes with the event rather than through a chain of template
    // refs: the gesture starts on something inside the drawing, and that is the
    // shortest honest way to the drawing itself.
    const element = (event.target as SVGElement | null)?.ownerSVGElement ?? null
    if (!element) return
    surface.value = element

    const began = positionIn(element, event)
    if (!began) return

    from.value = source
    start.value = began
    at.value = began
    pointer.value = event.pointerId

    // Capture keeps the gesture attached to the drawing once the pointer has
    // left it. It is allowed to fail — a pointer that is not actually down
    // cannot be captured — and failing must not take the listeners below with
    // it, or the gesture would begin and never end.
    try {
      element.setPointerCapture?.(event.pointerId)
    } catch {
      // Nothing to hold on to; the listeners below still follow it.
    }

    // Followed on the window rather than on the drawing. A gesture is let go
    // of wherever the hand happens to be, which is often past the edge of the
    // plex, and capture is a courtesy the browser may decline.
    //
    // One pointer at a time: a second finger arriving is not this gesture, and
    // its release is not this gesture's release.
    const isMine = (event: PointerEvent) => event.pointerId === pointer.value

    const onMove = (event: PointerEvent) => {
      if (isMine(event)) move(event)
    }
    const onUp = (up: PointerEvent) => {
      if (isMine(up)) finish()
    }
    // The browser takes the pointer away — a drag the system turned into a
    // scroll, a pen lifted out of range. No `pointerup` follows, so without
    // this the thread stays drawn and the next release anywhere makes a node.
    const onLost = (event: PointerEvent) => {
      if (isMine(event)) cancel()
    }
    const onKey = (key: KeyboardEvent) => {
      if (key.key === 'Escape') cancel()
    }

    detach = () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      window.removeEventListener('pointercancel', onLost)
      window.removeEventListener('keydown', onKey)
    }

    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    window.addEventListener('pointercancel', onLost)
    window.addEventListener('keydown', onKey)
  }

  // A plex can go while a hand is still on it.
  onScopeDispose(() => stop(svg()))

  return { from, at, outcome, begin, ask, cancel }
}
