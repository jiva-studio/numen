import type { ConnectionTypeInfo, RowDraft } from '@/entities/connection'
import type {
  InstallationConnections,
  ListModels,
  ListReasoningLevels,
  SelectChoice,
} from '../../../types'

/** What a row of a setting of rows is drawn from. */
export interface ConnectionRowEditorProps {
  row: RowDraft
  position: number
  count: number
  operation: string
  choices: readonly SelectChoice[]
  reason: string
  /** Whether the setting holds one row: the row cannot be moved or removed. */
  isSingle: boolean
  isHeld: boolean
  isDragging: boolean
  isDropTarget: boolean
  /** The types the dialog of a new connection offers. */
  types: readonly ConnectionTypeInfo[]
  create: InstallationConnections['create']
  update: InstallationConnections['update']
  listModels?: ListModels
  listReasoningLevels?: ListReasoningLevels
}
