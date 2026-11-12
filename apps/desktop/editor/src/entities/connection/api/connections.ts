/**
 * The calls of the connections: the kinds of connection the plugins provide, the connections the
 * person made, and the models a connection lists. A call that cannot reach the core rejects.
 */
import { connections, models } from '@/shared/api/clients'
import type { Result } from '@/shared/lib/result'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionFailure,
  ConnectionInfo,
  ConnectionTypeInfo,
  ModelInfo,
  ModelListFailure,
  ReasoningLevelInfo,
} from '../lib/types'

/** Every kind of connection the installed plugins provide, available or not. */
export const listConnectionTypes = async (): Promise<readonly ConnectionTypeInfo[]> =>
  (await connections.listTypes()).map((one) => ({ ...one }))

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

/** The models a connection offers for an operation, from the provider of its type. */
export const listModels = async (
  connectionId: string,
  operation: string,
): Promise<Result<readonly ModelInfo[], ModelListFailure>> =>
  models.listModels(connectionId, operation)

/** The levels of reasoning effort the provider lets a model run at; none where it has no such setting. */
export const listReasoningLevels = async (
  connectionId: string,
  model: string,
): Promise<Result<readonly ReasoningLevelInfo[], ModelListFailure>> =>
  models.listReasoningLevels(connectionId, model)
