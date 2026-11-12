import type { ConnectionFieldInfo, ConnectionTypeInfo, RowDraft } from '@/entities/connection'
import type { InstallationConnections, ListRowOptions, SelectChoice } from '../../../types'

/** What a row of a setting of rows is drawn from. */
export interface ConnectionRowEditorProps {
  row: RowDraft
  position: number
  count: number
  /** The operations the setting of rows is used for. */
  operations: readonly string[]
  choices: readonly SelectChoice[]
  reason: string
  /** The row fields of the type of the connection the row names. */
  rowFields: readonly ConnectionFieldInfo[]
  /** Whether the setting holds one row: the row cannot be moved or removed. */
  isSingle: boolean
  isHeld: boolean
  isDragging: boolean
  isDropTarget: boolean
  /** The types the dialog of a new connection offers. */
  types: readonly ConnectionTypeInfo[]
  create: InstallationConnections['create']
  update: InstallationConnections['update']
  listRowOptions?: ListRowOptions
}
