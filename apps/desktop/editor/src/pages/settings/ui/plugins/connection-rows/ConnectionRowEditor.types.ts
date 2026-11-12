import type { ConnectionFieldInfo, ConnectionTypeInfo, RowDraft } from '@/entities/connection'
import type { InstallationConnections, ListRowOptions, SelectChoice } from '../../../types'

/**
 * What a row of a setting of rows is drawn from. A row of a setting that holds one row (isSingle) cannot be
 * moved or removed. types are the types the dialog of a new connection offers.
 */
export interface ConnectionRowEditorProps {
  row: RowDraft
  position: number
  count: number
  operations: readonly string[]
  choices: readonly SelectChoice[]
  reason: string
  rowFields: readonly ConnectionFieldInfo[]
  isSingle: boolean
  isHeld: boolean
  isDragging: boolean
  isDropTarget: boolean
  types: readonly ConnectionTypeInfo[]
  create: InstallationConnections['create']
  update: InstallationConnections['update']
  listRowOptions?: ListRowOptions
}
