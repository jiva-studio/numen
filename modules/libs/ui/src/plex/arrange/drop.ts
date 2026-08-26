import type { PlacedNode, PlexFrame, PlexRelatedSeat, Point } from '../model'
import { RELATED_SEATS } from '../model'
import type { Direction, PlexOptions, Size } from './options'

/**
 * What dragging away from a node and letting go comes to.
 *
 * The plex works out the shape of the gesture and nothing else: which node it
 * started from, which way it went, and whether it landed on something. What a
 * parent or a jump then *means* — what gets written, and whether it is allowed
 * — is the application's, which is why this reports rather than acts.
 */
export type Drop =
  | { readonly kind: 'create'; readonly from: string; readonly seat: PlexRelatedSeat }
  | {
      readonly kind: 'link'
      readonly from: string
      readonly to: string
      readonly seat: PlexRelatedSeat
    }

/**
 * The seat a gesture that went nowhere asks for.
 *
 * One more child, because that is what anyone reaches for; failing that, the
 * first seat the caller allows. A gesture goes nowhere when it is a press
 * rather than a drag — from a trackpad, from a hand that cannot hold a button
 * down, or from the keyboard, where there is no direction to read at all.
 */
export const seatWithoutDirection = (
  allowed: readonly PlexRelatedSeat[],
): PlexRelatedSeat | null =>
  allowed.includes('child') ? 'child' : (allowed[0] ?? null)

/**
 * Which way a point lies from another.
 *
 * Not simply the axis it lies furthest along: rows are wide and columns are
 * narrow, so the wedge meaning up or down is wider than a quarter turn. At a
 * bias of one, the outermost child of a wide row is further sideways than it
 * is down, and dragging towards where the children plainly are would name
 * something off to the side.
 */
function towards(dx: number, dy: number, bias: number): Direction | null {
  if (dx === 0 && dy === 0) return null
  return Math.abs(dy) * bias >= Math.abs(dx)
    ? dy < 0
      ? 'up'
      : 'down'
    : dx < 0
      ? 'left'
      : 'right'
}

/**
 * The seat a direction stands for — the inverse of the arrangement's own map.
 *
 * Read off `direction` rather than assumed, because that is what put the seats
 * where they are: with parents sent down, dragging down means a parent, and a
 * rule written the other way would quietly contradict the drawing.
 */
export function seatTowards(
  from: Point,
  to: Point,
  options: PlexOptions,
): PlexRelatedSeat | null {
  const heading = towards(to.x - from.x, to.y - from.y, options.gesture.verticalBias)
  if (!heading) return null
  return RELATED_SEATS.find((seat) => options.direction[seat] === heading) ?? null
}

/**
 * The node a point falls inside, if any. Later nodes win, as when drawn.
 *
 * A node that is not fully there is not a node to let go on: it is on its way
 * in or out, and it would be a link to something the reader never saw. That is
 * the same rule that decides the click and the tab stop.
 */
export function nodeAt(point: Point, frame: PlexFrame): PlacedNode | null {
  for (let i = frame.nodes.length - 1; i >= 0; i--) {
    const node = frame.nodes[i]
    if (!node || node.opacity < 1) continue
    if (
      Math.abs(point.x - node.x) <= node.width / 2 &&
      Math.abs(point.y - node.y) <= node.height / 2
    ) {
      return node
    }
  }
  return null
}

export interface DropInput {
  readonly frame: PlexFrame
  readonly options: PlexOptions
  /** The node the gesture started from. */
  readonly from: string
  /** Where it was let go, in the plex's own coordinates. */
  readonly at: Point
  /** Seats a gesture is allowed to produce. */
  readonly allowed: readonly PlexRelatedSeat[]
}

/**
 * Landing on another node makes a link; landing on nothing makes a node. That
 * one rule is also what keeps a gesture from a node other than the focus
 * workable: the seats around a child overlap the rest of the picture, and
 * whatever the gesture crosses, only where it stops decides.
 */
export function resolveDrop({
  frame,
  options,
  from,
  at,
  allowed,
}: DropInput): Drop | null {
  const source = frame.nodes.find((node) => node.id === from)
  if (!source) return null

  const landedOn = nodeAt(at, frame)
  if (landedOn?.id === from) return null

  const towardsPoint = landedOn ? { x: landedOn.x, y: landedOn.y } : at
  const seat = seatTowards(source, towardsPoint, options)
  if (!seat || !allowed.includes(seat)) return null

  return landedOn
    ? { kind: 'link', from, to: landedOn.id, seat }
    : { kind: 'create', from, seat }
}

export interface CarriedInput {
  readonly frame: PlexFrame
  readonly options: PlexOptions
  /** The window the plex is drawn in, centred on the focus. */
  readonly viewport: Size
  /** Where the pointer is, in the plex's own coordinates. */
  readonly at: Point
  /** Seats a gesture is allowed to produce. */
  readonly allowed: readonly PlexRelatedSeat[]
  /** How far from the focus the pointer stands before it names a direction. */
  readonly threshold: number
}

/**
 * The seat something carried in from outside comes to.
 *
 * Measured from the focus, which is what the arrangement is built around. A
 * node the pointer crosses is not a landing: the seat is read off the
 * direction, and letting go anywhere in the window is answered the same way.
 */
export function seatCarried({
  frame,
  options,
  viewport,
  at,
  allowed,
  threshold,
}: CarriedInput): PlexRelatedSeat | null {
  const focus = frame.nodes.find((node) => node.seat === 'focus')
  if (!focus) return null

  // Past the edge of the window is over whatever is drawn beside the plex.
  if (Math.abs(at.x) > viewport.width / 2 || Math.abs(at.y) > viewport.height / 2) return null

  // Close enough to the focus that the direction is the tremor of a hand.
  if (Math.hypot(at.x - focus.x, at.y - focus.y) < threshold) return null

  const seat = seatTowards(focus, at, options)
  return seat && allowed.includes(seat) ? seat : null
}
