/**
 * The calls of the connections: the kinds of connection the plugins provide, the connections the
 * person made, and the options a row field of a connection lists. A call that cannot reach the core rejects.
 */
import { connections } from '@/shared/api/clients'
import type { Result } from '@/shared/lib/result'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionFailure,
  ConnectionInfo,
  ConnectionTypeList,
  RowOption,
} from '../lib/types'

/**
 * Every kind of connection the installed plugins provide, available or not, and the titles of the
 * operations; the first title of a name counts.
 */
export const listConnectionTypes = async (): Promise<ConnectionTypeList> => {
  const { types, operations } = await connections.listTypes()
  const operationTitles: Record<string, string> = {}
  for (const { name, title } of operations) {
    if (title !== '' && !(name in operationTitles)) operationTitles[name] = title
  }
  return { types: types.map((one) => ({ ...one })), operationTitles }
}

/** Every connection, in the order it was made. */
export const listConnections = async (): Promise<readonly ConnectionInfo[]> =>
  (await connections.list()).map((one) => ({ ...one }))

export const createConnection = async (
  draft: ConnectionDraft,
): Promise<Result<ConnectionInfo, ConnectionFailure>> => connections.create(draft)

export const updateConnection = async (
  change: ConnectionChange,
): Promise<Result<ConnectionInfo, ConnectionFailure>> => connections.update(change)

export const deleteConnection = async (id: string): Promise<Result<undefined, ConnectionFailure>> =>
  connections.delete(id)

/**
 * The values a row field of the type of a connection can take, from the provider of its type, for the
 * values the row holds so far. `operations` are the operations the setting of rows is used for.
 */
export const listRowOptions = async (
  connectionId: string,
  field: string,
  options: Readonly<Record<string, string>>,
  operations: readonly string[],
): Promise<Result<readonly RowOption[], ConnectionFailure>> =>
  connections.listRowOptions(connectionId, field, options, operations)
