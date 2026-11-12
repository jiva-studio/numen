import type { RowDraft } from '@/entities/connection'
import type { ListModels, ListReasoningLevels, SelectChoice } from '../../../types'

/** What a row of a setting of rows is drawn from. */
export interface ConnectionRowEditorProps {
  row: RowDraft
  position: number
  count: number
  operation: string
  choices: readonly SelectChoice[]
  reason: string
  isHeld: boolean
  isDragging: boolean
  isDropTarget: boolean
  listModels?: ListModels
  listReasoningLevels?: ListReasoningLevels
}
