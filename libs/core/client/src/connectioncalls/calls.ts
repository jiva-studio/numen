/**
 * The operations a plugin view asks the core to carry out through the rows of the plugin. The core knows
 * an operation by its name only; the payload and the frames are bytes of the contract the operation
 * belongs to.
 */
import { createClient, type Client } from '@connectrpc/connect'
import { CallErrorCode, ConnectionCallService, Priority, type CallFailure } from '@numen/protocol'
import { asFailure, asValue, type Result } from '../result'
import { transport } from '../transport'

/** The calls of ConnectionCallService. */
export type ConnectionCallRpc = Pick<
  Client<typeof ConnectionCallService>,
  'call' | 'listRows' | 'listRowOptions'
>

/** Whose action a call serves. A call that names none is background. */
export type ConnectionCallPriority = 'interactive' | 'background'

/**
 * A connection among the rows of the plugin for the operation, asked before the other rows with `options`
 * in place of the values of its row fields. A connection that is not among the rows is ignored.
 */
export interface ConnectionCallPreferredRow {
  readonly connectionId: string
  readonly options: Readonly<Record<string, string>>
}

/** What a call may carry besides its operation and payload. */
export interface ConnectionCallOptions {
  readonly signal?: AbortSignal | undefined
  readonly priority?: ConnectionCallPriority | undefined
  readonly preferred?: ConnectionCallPreferredRow | undefined
}

/** An operation a plugin asks for, with the payload in the bytes of its contract. */
export interface ConnectionCallRequest {
  readonly pluginId: string
  readonly operation: string
  readonly payload: Uint8Array
}

/** One row of a plugin that did not answer. */
export interface ConnectionCallRowFailure {
  readonly connectionName: string
  readonly reason: string
}

/** Why no row answered a call. */
export type ConnectionCallErrorCode = 'no_connection' | 'all_failed' | 'stopped'

/** A served refusal of a call, with the reason the core gives and the rows that failed. */
export interface ConnectionCallFailure {
  readonly errorCode: ConnectionCallErrorCode
  readonly detail: string
  readonly failures: readonly ConnectionCallRowFailure[]
}

/** A row field a connection type declares. `optionsFrom` is the operation that lists its values, empty for none. */
export interface ConnectionCallRowField {
  readonly key: string
  readonly title: string
  readonly optionsFrom: string
}

/** One row of a plugin: its connection, the values of its row fields and the row fields of its type. */
export interface ConnectionCallRow {
  readonly connectionId: string
  readonly connectionName: string
  readonly options: Readonly<Record<string, string>>
  readonly fields: readonly ConnectionCallRowField[]
}

/** One value a row field can take. An empty title shows the value, and an empty group no heading. */
export interface ConnectionCallRowOption {
  readonly value: string
  readonly title: string
  readonly group: string
}

/** A row field of a connection among the rows of a plugin, with the values of the row fields so far. */
export interface ConnectionCallRowOptionsRequest {
  readonly pluginId: string
  readonly operation: string
  readonly connectionId: string
  readonly field: string
  readonly options: Readonly<Record<string, string>>
}

/** The calls of the plugins. A call that cannot reach the core throws. */
export interface ConnectionCallClient {
  /**
   * Carries out the operation through the rows of the plugin and yields its data frames. The generator
   * returns the done payload or the failure the core serves, and throws where the stream ends before the
   * call did.
   */
  call(
    request: ConnectionCallRequest,
    options?: ConnectionCallOptions,
  ): AsyncGenerator<Uint8Array, Result<Uint8Array, ConnectionCallFailure>>
  /**
   * The rows of the plugin for the operation in the order of use, without the rows whose connection is
   * missing or whose type is unavailable or does not serve the operation.
   */
  listRows(pluginId: string, operation: string): Promise<readonly ConnectionCallRow[]>
  /**
   * The values a row field of a connection among the rows can take, as its provider lists them. It throws
   * for a connection that is not among the rows and for a provider that cannot list them.
   */
  listRowOptions(
    request: ConnectionCallRowOptionsRequest,
  ): Promise<readonly ConnectionCallRowOption[]>
}

const readErrorCode = (code: CallErrorCode): ConnectionCallErrorCode => {
  switch (code) {
    case CallErrorCode.NO_CONNECTION:
      return 'no_connection'
    case CallErrorCode.ALL_FAILED:
      return 'all_failed'
    case CallErrorCode.STOPPED:
      return 'stopped'
    case CallErrorCode.UNSPECIFIED:
      throw new Error('ConnectionCallService ended a call with a failure that holds no error')
  }
}

const readFailure = (failure: CallFailure): ConnectionCallFailure => ({
  errorCode: readErrorCode(failure.error),
  detail: failure.detail,
  failures: failure.failures.map((one) => ({
    connectionName: one.connectionName,
    reason: one.reason,
  })),
})

const writePriority = (priority: ConnectionCallPriority | undefined): Priority =>
  priority === 'interactive' ? Priority.INTERACTIVE : Priority.BACKGROUND

const writePreferredRow = (preferred: ConnectionCallPreferredRow | undefined) =>
  preferred === undefined
    ? {}
    : { preferred: { connectionId: preferred.connectionId, options: { ...preferred.options } } }

/** The calls carried out through a ConnectionCallService client. */
export const createConnectionCallClient = (rpc: ConnectionCallRpc): ConnectionCallClient => ({
  call: async function* (request, options) {
    const stream = rpc.call(
      {
        ...request,
        priority: writePriority(options?.priority),
        ...writePreferredRow(options?.preferred),
      },
      options?.signal === undefined ? {} : { signal: options.signal },
    )
    for await (const answer of stream) {
      switch (answer.frame.case) {
        case 'data':
          yield answer.frame.value
          break
        case 'done':
          return asValue(answer.frame.value)
        case 'failure':
          return asFailure(readFailure(answer.frame.value))
        case undefined:
          break
      }
    }
    throw new Error('ConnectionCallService ended the stream before the call did')
  },
  listRows: async (pluginId, operation) => {
    const { rows } = await rpc.listRows({ pluginId, operation })
    return rows.map((row) => ({
      connectionId: row.connectionId,
      connectionName: row.connectionName,
      options: { ...row.options },
      fields: row.fields.map(({ key, title, optionsFrom }) => ({ key, title, optionsFrom })),
    }))
  },
  listRowOptions: async (request) => {
    const { options } = await rpc.listRowOptions({ ...request, options: { ...request.options } })
    return options.map(({ value, title, group }) => ({ value, title, group }))
  },
})

/** The calls of the window the page is served from. */
export const connectionCalls = createConnectionCallClient(
  createClient(ConnectionCallService, transport),
)
