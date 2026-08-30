/**
 * What dragging a card comes to.
 *
 * Apart from the component because how far is far enough is a rule, and a rule
 * inside a component can only be exercised by dragging something.
 */

/** A press that moved less than this across is a press, and turns the card. */
export const STILL = 4

/** How far the card is dragged before the panel comes or goes, in pixels. */
export const FAR = 64

/** What a finished drag asks for, and nothing when it asks nothing. */
export type Asks = { does: 'open' } | { does: 'shut' } | { does: 'press' } | null

/** A drag as it ended: how far it went, and whether the panel was already up. */
export interface Dragged {
  /** How far across the card was taken, negative towards the panel. */
  readonly moved: number
  readonly open: boolean
}

/**
 * The card is taken to the left to bring the panel in, and to the right to send
 * it away. A drag the other way while the panel is where it asks for is
 * nothing, so a hand that overshoots and comes back asks for nothing twice.
 */
export function swiped({ moved, open }: Dragged): Asks {
  if (Math.abs(moved) <= STILL) return { does: 'press' }
  if (!open && moved <= -FAR) return { does: 'open' }
  if (open && moved >= FAR) return { does: 'shut' }
  return null
}

/**
 * How far the panel is drawn while the hand is still on the card, from nothing
 * to the whole of it. The panel follows the hand, so a drag that stops halfway
 * leaves it halfway until the hand lets go.
 */
export const drawn = ({ moved, open }: Dragged): number => {
  const across = open ? FAR - moved : -moved
  return Math.min(1, Math.max(0, across / FAR))
}
