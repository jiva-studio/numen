import type { Stencil } from '../../lib/card'
import type { DeckProblems, DeckWords } from '../../lib/deck'
import type { Run } from '../../lib/grid'
import type { InsertionPoint } from '../../lib/order'

export interface DeckRunProps {
  /** The run, as the grid lays it out. */
  run: Run
  /** The stencils a card may be cut by. */
  stencils: readonly Stencil[]
  /** How many stand in the grid, every plus among them. */
  of: number
  /** What the caller found wrong with the cards it handed in. */
  wrong: DeckProblems
  /** Where letting go would put the card being dragged. */
  at: InsertionPoint | undefined
  /** Its plus is showing which stencils a new card may be cut by. */
  isAsking: boolean
  /** The words it is drawn with. */
  words?: DeckWords
}
