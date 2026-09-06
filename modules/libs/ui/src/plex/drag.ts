/**
 * Something dragged over the plex from outside it. Everything that knows about
 * events and screen pixels is here; which seat a point comes to is worked out
 * in `arrange/drop.ts`, as a value.
 */
import { computed, onScopeDispose, ref, watch, type Ref } from 'vue'
import { seatDropped, type PlexOptions, type Size } from './arrange'
import { pointIn } from './gesture'
import type { PlexFrame } from './frame'
import type { Point } from './node'
import type { PlexRelatedSeat } from './seat'

const CAPTURE = { capture: true } as const

export interface PlexDragState {
  /** Where the pointer is, in the plex's own coordinates. */
  readonly at: Ref<Point | null>
  /** The seat letting go here comes to, so it can be shown before it does. */
  readonly seat: Ref<PlexRelatedSeat | null>
}

/** What following a drag takes: the drawing, the picture, and the rules. */
export interface PlexDragDeps {
  /** The drawing, which turns screen pixels into the plex's own coordinates. */
  readonly surface: () => SVGSVGElement | null
  /** What is being dragged, each of them opaque. Empty while nothing is. */
  readonly dragged: () => readonly string[]
  readonly frame: () => PlexFrame
  readonly options: () => PlexOptions
  readonly viewport: () => Size
  /** Seats a gesture is allowed to produce. */
  readonly allowed: () => readonly PlexRelatedSeat[]
  /** How far from the focus the pointer stands before it names a direction. */
  readonly threshold: () => number
  readonly settle: (dragged: readonly string[], seat: PlexRelatedSeat) => void
}

/**
 * Follow a pointer dragging something across the plex until it is let go or
 * given up on.
 *
 * The gesture began somewhere the plex cannot see, so it is followed on the
 * window for as long as there is something to drag, and it is over the
 * instant the pointer comes up wherever that is.
 */
export function usePlexDrag(drag: PlexDragDeps): PlexDragState {
  /** Where the pointer is, and nothing at all while nothing is being dragged. */
  const at = ref<Point | null>(null)

  const seat = computed<PlexRelatedSeat | null>(() => {
    const point = at.value
    if (!point) return null
    return seatDropped({
      frame: drag.frame(),
      options: drag.options(),
      viewport: drag.viewport(),
      at: point,
      allowed: drag.allowed(),
      threshold: drag.threshold(),
    })
  })

  /** What the drag under way installed on the window, if anything. */
  let detach: (() => void) | null = null

  /**
   * What the drag was handed, held for the life of the gesture. Whoever is
   * dragging them may put them down on the same release this settles on.
   */
  let holding: readonly string[] = []

  const stop = () => {
    detach?.()
    detach = null
    at.value = null
    holding = []
  }

  const move = (event: PointerEvent) => {
    const element = drag.surface()
    at.value = element ? pointIn(element, event) : null
  }

  const finish = (event: PointerEvent) => {
    move(event)
    const dragged = holding
    const settled = seat.value
    stop()
    if (dragged.length > 0 && settled) drag.settle(dragged, settled)
  }

  const follow = () => {
    if (detach) return
    holding = drag.dragged()

    const onMove = (moved: PointerEvent) => move(moved)
    const onUp = (up: PointerEvent) => finish(up)
    // The browser takes the pointer away, and no `pointerup` follows.
    const onLost = () => stop()
    const onKey = (key: KeyboardEvent) => {
      if (key.key === 'Escape') stop()
    }

    detach = () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp, CAPTURE)
      window.removeEventListener('pointercancel', onLost)
      window.removeEventListener('keydown', onKey)
    }

    window.addEventListener('pointermove', onMove)
    // The release is answered on its way down the page, ahead of whoever is
    // dragging them and puts them down on the way back up.
    window.addEventListener('pointerup', onUp, CAPTURE)
    window.addEventListener('pointercancel', onLost)
    window.addEventListener('keydown', onKey)
  }

  watch(
    () => drag.dragged().length > 0,
    (dragging) => (dragging ? follow() : stop()),
    { immediate: true },
  )

  // A plex can go while something is still being dragged over it.
  onScopeDispose(stop)

  return { at, seat }
}
