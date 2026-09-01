/**
 * What a menu is, as plain values. No DOM, no measurement, no clock.
 */
import { beside } from '../placing/place'
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
  /**
   * The band it belongs to. Items of one band stand together, and a rule is
   * drawn where one band gives way to the next. A menu whose items name no
   * band is one band and carries no rules.
   */
  readonly band?: string
}

/** One item as it is drawn: what it says, and the rule standing above it. */
export interface BandedItem extends MenuItem {
  /** It begins a band, and a rule stands between it and what is above. */
  readonly rule: boolean
}

/**
 * The items in the order they were given, each saying whether a rule stands
 * above it. The first item begins the menu, and nothing is drawn above it.
 */
export const banded = (items: readonly MenuItem[]): readonly BandedItem[] =>
  items.map((item, at) => ({ ...item, rule: at > 0 && item.band !== items[at - 1]?.band }))

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
 * Where a menu of this size, asked for at this point, is drawn.
 *
 * The point is a span of no width, touching the menu: the menu runs on from it
 * and folds back over it at an edge. Wider than the area it is placed in, it
 * sits at the near edge and scrolls.
 */
export const placeMenu = ({ at, size, viewport, margin }: MenuPlacement): MenuPlacing => ({
  x: beside({
    from: at.x,
    to: at.x,
    size: size.width,
    room: viewport.width,
    margin,
    gap: 0,
  }),
  y: beside({
    from: at.y,
    to: at.y,
    size: size.height,
    room: viewport.height,
    margin,
    gap: 0,
  }),
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

/**
 * How a menu came to be open, declared once. Where the keyboard is as it
 * appears is derived from this table.
 */
export interface MenuOpeningDescriptor {
  /** Whether the keyboard lands on an item as the menu appears. */
  readonly lands: boolean
}

export const MENU_OPENINGS = {
  pointer: { lands: false },
  keyboard: { lands: true },
} as const satisfies Record<string, MenuOpeningDescriptor>

/** What opened a menu: a hand, or the keyboard. */
export type MenuOpening = keyof typeof MENU_OPENINGS

export const MENU_OPENINGS_ALL = Object.keys(MENU_OPENINGS) as readonly MenuOpening[]

/**
 * Where the keyboard is as a menu opens. Opened by hand it is on no item, and
 * the first step down from there lands on the first one. Opened by the
 * keyboard it lands on the item in force, and on the first where the menu
 * holds none.
 */
export const landsOn = (
  opening: MenuOpening,
  items: readonly MenuItem[],
  current: string | null = null,
): number => {
  if (!MENU_OPENINGS[opening].lands) return -1
  const at = items.findIndex((item) => item.id === current && !item.disabled)
  return at >= 0 ? at : stepTo(items, -1, 1)
}
