/**
 * What a menu is, as plain values. No DOM, no measurement, no clock.
 */
import type { Point } from '../plex/model'
import type { Size } from '../plex/arrange'

/**
 * One thing that can be chosen. The identifier is opaque: the menu has no way
 * to ask what it addresses, and hands it back as given.
 */
export interface MenuItem {
  readonly id: string
  /** What is written on it. */
  readonly text: string
  /** Drawn and announced, and not choosable. */
  readonly disabled?: boolean
}

/** What placing a menu needs to know. */
export interface MenuPlacement {
  /** Where it was asked for. */
  readonly at: Point
  /** How big it turned out to be. */
  readonly size: Size
  /** The area it is placed in. */
  readonly viewport: Size
  /** Kept clear of that area's edges, so nothing sits flush against them. */
  readonly margin: number
}

/** Where the menu goes, in the coordinates the point arrived in. */
export interface MenuPlacing {
  readonly x: number
  readonly y: number
}

/**
 * One axis.
 *
 * A menu runs on from the point it was asked for. Where the far edge is nearer
 * than its own length it runs back over the point instead, and either way it
 * is brought inside the edges it may touch. Wider than the area it is placed
 * in, it sits at the near edge and scrolls.
 */
const along = (at: number, size: number, room: number, margin: number): number => {
  const back = at - size
  const start = at + size + margin <= room || back < margin ? at : back
  return Math.max(margin, Math.min(start, room - size - margin))
}

/** Where a menu of this size, asked for at this point, is drawn. */
export const placeMenu = ({ at, size, viewport, margin }: MenuPlacement): MenuPlacing => ({
  x: along(at.x, size.width, viewport.width, margin),
  y: along(at.y, size.height, viewport.height, margin),
})

/**
 * Where the keyboard lands next, counting from `from` and passing over the
 * items that cannot be chosen. It wraps, and answers -1 when there is nothing
 * to land on.
 *
 * Counting from -1 by one is how the first is asked for, and from 0 by minus
 * one is how the last is.
 */
export const stepTo = (items: readonly MenuItem[], from: number, by: number): number => {
  const total = items.length
  for (let step = 1; step <= total; step += 1) {
    const at = (((from + by * step) % total) + total) % total
    if (!items[at]?.disabled) return at
  }
  return -1
}
