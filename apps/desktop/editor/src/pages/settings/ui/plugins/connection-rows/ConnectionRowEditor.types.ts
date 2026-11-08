import type { SelectChoice } from '@numen/ui'
import type { ModelInfo, ModelListFailure, ReasoningLevelInfo, Result } from '@numen/core-client'
import type { RowDraft } from '@/entities/connection'

/** Asks the provider of a connection for the models that fit an operation. */
export type ListModels = (
  connectionId: string,
  operation: string,
) => Promise<Result<readonly ModelInfo[], ModelListFailure>>

/** Asks the provider of a connection for the reasoning levels of a model. */
export type ListReasoningLevels = (
  connectionId: string,
  model: string,
) => Promise<Result<readonly ReasoningLevelInfo[], ModelListFailure>>

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
