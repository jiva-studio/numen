import type { CardModel } from '../../../../lib/cards'
import type { DeckProblems, DeckWords } from '../../../../lib/cards'
import type { Run } from '../../../../lib/cards'
import type { InsertionPoint } from '../../../../lib/cards'

export interface DeckRunProps {
  run: Run
  cardModels: readonly CardModel[]
  of: number
  wrong: DeckProblems
  at: InsertionPoint | undefined
  isAsking: boolean
  words?: DeckWords
}
