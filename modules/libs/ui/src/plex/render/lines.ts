/** An edge with what the drawing asks of it, worked out once for both layers. */
import type { PlacedEdge } from '../edge'

export interface EdgeLine {
  readonly edge: PlacedEdge
  /** One drawing per pair and direction, so a pair may carry two lines. */
  readonly key: string
  /** What the hand is on: two lines between one pair are one line to point at. */
  readonly pair: string
  readonly d: string
  /** The head on its end of the line, and nothing for a line carrying none. */
  readonly arrow: string | null
  /** The path the title is set along, named, and nothing for a line with no title. */
  readonly titlePath: string | null
  readonly titleLine: string
  /** How far along that path the middle of the title stands. */
  readonly titleAt: string
}
