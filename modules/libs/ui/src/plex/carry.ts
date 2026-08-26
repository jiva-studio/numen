/**
 * Something carried over the plex from outside it. Everything that knows about
 * events and screen pixels is here; which seat a point comes to is worked out
 * in `arrange/drop.ts`, as a value.
 */
import { computed, onScopeDispose, ref, watch, type Ref } from 'vue'
import { seatCarried, type PlexOptions, type Size } from './arrange'
import { pointIn } from './gesture'
import type { PlexFrame, PlexRelatedSeat, Point } from './model'

const CAPTURE = { capture: true } as const

export interface Carrying {
  /** Where the pointer is, in the plex's own coordinates. */
  readonly at: Ref<Point | null>
  /** The seat letting go here comes to, so it can be shown before it does. */
  readonly seat: Ref<PlexRelatedSeat | null>
}

/** What following a carry takes: the drawing, the picture, and the rules. */
export interface Carry {
  /** The drawing, which turns screen pixels into the plex's own coordinates. */
  readonly surface: () => SVGSVGElement | null
  /** What is being carried, each of them opaque. Empty while nothing is. */
  readonly carried: () => readonly string[]
  readonly frame: () => PlexFrame
  readonly options: () => PlexOptions
  readonly viewport: () => Size
  /** Seats a gesture is allowed to produce. */
  readonly allowed: () => readonly PlexRelatedSeat[]
  /** How far from the focus the pointer stands before it names a direction. */
  readonly threshold: () => number
  readonly settle: (carried: readonly string[], seat: PlexRelatedSeat) => void
}

/**
 * Follow a pointer carrying something across the plex until it is let go or
 * given up on.
 *
 * The gesture began somewhere the plex cannot see, so it is followed on the
 * window for as long as there is something to carry, and it is over the
 * instant the pointer comes up wherever that is.
 */
export function usePlexCarry(carry: Carry): Carrying {
  /** Where the pointer is, and nothing at all while nothing is being carried. */
  const at = ref<Point | null>(null)

  const seat = computed<PlexRelatedSeat | null>(() => {
    const point = at.value
    if (!point) return null
    return seatCarried({
      frame: carry.frame(),
      options: carry.options(),
      viewport: carry.viewport(),
      at: point,
      allowed: carry.allowed(),
      threshold: carry.threshold(),
    })
  })

  /** What the carry under way installed on the window, if anything. */
  let detach: (() => void) | null = null

  /**
   * What the carry was handed, held for the life of the gesture. Whoever is
   * carrying them may put them down on the same release this settles on.
   */
  let holding: readonly string[] = []

  const stop = () => {
    detach?.()
    detach = null
    at.value = null
    holding = []
  }

  const move = (event: PointerEvent) => {
    const element = carry.surface()
    at.value = element ? pointIn(element, event) : null
  }

  const finish = (event: PointerEvent) => {
    move(event)
    const carried = holding
    const settled = seat.value
    stop()
    if (carried.length > 0 && settled) carry.settle(carried, settled)
  }

  const follow = () => {
    if (detach) return
    holding = carry.carried()

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
    // carrying them and puts them down on the way back up.
    window.addEventListener('pointerup', onUp, CAPTURE)
    window.addEventListener('pointercancel', onLost)
    window.addEventListener('keydown', onKey)
  }

  watch(
    () => carry.carried().length > 0,
    (carrying) => (carrying ? follow() : stop()),
    { immediate: true },
  )

  // A plex can go while something is still being carried over it.
  onScopeDispose(stop)

  return { at, seat }
}
