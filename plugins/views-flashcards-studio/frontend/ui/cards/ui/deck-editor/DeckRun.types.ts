import type { CardModel } from '../../lib/card'
import type { DeckProblems, DeckWords } from '../../lib/deck'
import type { Run } from '../../lib/grid'
import type { InsertionPoint } from '../../lib/order'

export interface DeckRunProps {
  run: Run
  cardModels: readonly CardModel[]
  of: number
  wrong: DeckProblems
  at: InsertionPoint | undefined
  isAsking: boolean
  words?: DeckWords
}
