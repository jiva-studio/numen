import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionFieldInfo,
  ConnectionInfo,
  ConnectionTypeInfo,
  RowDraft,
  RowOption,
} from '@/entities/connection'
import type { Result } from '@/shared/lib/result'
import type { SelectChoice } from '../../../types'

/*
 * The calls are written as function types: Vue reads an indexed method signature such as
 * InstallationConnections['create'] as its return type and checks the prop for a Promise.
 */

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
  create: (draft: ConnectionDraft) => Promise<Result<ConnectionInfo, string>>
  update: (change: ConnectionChange) => Promise<Result<ConnectionInfo, string>>
  listRowOptions?: (
    connectionId: string,
    field: string,
    options: Readonly<Record<string, string>>,
    operations: readonly string[],
  ) => Promise<Result<readonly RowOption[], string>>
}
