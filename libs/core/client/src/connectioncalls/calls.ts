/**
 * The operations a plugin view asks the core to carry out through the rows of the plugin. The core knows
 * an operation by its name only; the payload and the frames are bytes of the contract the operation
 * belongs to.
 */
import type { Client } from '@connectrpc/connect'
import {
  ErrorCode,
  type ConnectionCallService,
  FailureReason,
  Priority,
  type CallFailure,
} from '@numen/protocol'
import type { RowOption } from '../connections'
import { asFailure, asValue, type Failure, type Result } from '@numen/result'

/** The calls of ConnectionCallService. */
export type ConnectionCallRpc = Pick<
  Client<typeof ConnectionCallService>,
  'call' | 'cancelCall' | 'listRows' | 'listRowOptions'
>

/** Whose action a call serves. A call that names none is background. */
export type ConnectionCallPriority = 'interactive' | 'background'

/**
 * A connection among the rows of the plugin for the operation, asked before the other rows with `options`
 * in place of the values of its row fields. A connection that is not among the rows is ignored.
 * `isExclusive` asks that row alone, and no row where the connection is not among the rows.
 */
export interface ConnectionCallPreferredRow {
  readonly connectionId: string
  readonly options: Readonly<Record<string, string>>
  readonly isExclusive?: boolean | undefined
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

/**
 * Why a row did not answer, as a program reads it. A row the core set aside for a reason outside these
 * has `unavailable`.
 */
export type ConnectionCallFailureCode =
  'unavailable' | 'rejectedKey' | 'paymentRequired' | 'rateLimited' | 'invalidRequest'

/** One row of a plugin that did not answer: `reason` is the words the person reads, `code` the reason. */
export interface ConnectionCallRowFailure {
  readonly connectionName: string
  readonly reason: string
  readonly code: ConnectionCallFailureCode
}

/** Why no row answered a call. */
export type ConnectionCallErrorCode = 'noConnection' | 'allFailed' | 'stopped'

/** A served refusal of a call, with the reason the core gives and the rows that failed. */
export interface ConnectionCallFailure extends Failure<ConnectionCallErrorCode> {
  readonly failures: readonly ConnectionCallRowFailure[]
}

/**
 * A row field a connection type declares. `optionsFrom` is the operation that lists its values, empty
 * for none. `isRequired` is a field a row must hold a value for.
 */
export interface ConnectionCallRowField {
  readonly key: string
  readonly title: string
  readonly optionsFrom: string
  readonly isRequired: boolean
}

/** One row of a plugin: its connection, the values of its row fields and the row fields of its type. */
export interface ConnectionCallRow {
  readonly connectionId: string
  readonly connectionName: string
  readonly options: Readonly<Record<string, string>>
  readonly fields: readonly ConnectionCallRowField[]
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
  ): Promise<readonly RowOption[]>
}

const readErrorCode = (code: ErrorCode): ConnectionCallErrorCode => {
  switch (code) {
    case ErrorCode.NO_CONNECTION:
      return 'noConnection'
    case ErrorCode.ALL_FAILED:
      return 'allFailed'
    case ErrorCode.STOPPED:
      return 'stopped'
    default:
      throw new Error('ConnectionCallService ended a call with a failure that holds no error')
  }
}

const readFailureCode = (code: FailureReason): ConnectionCallFailureCode => {
  switch (code) {
    case FailureReason.REJECTED_KEY:
      return 'rejectedKey'
    case FailureReason.PAYMENT_REQUIRED:
      return 'paymentRequired'
    case FailureReason.RATE_LIMITED:
      return 'rateLimited'
    case FailureReason.INVALID_REQUEST:
      return 'invalidRequest'
    case FailureReason.UNAVAILABLE:
    case FailureReason.UNSPECIFIED:
      return 'unavailable'
  }
}

const readFailure = (failure: CallFailure): ConnectionCallFailure => ({
  error: readErrorCode(failure.error),
  message: failure.detail,
  failures: failure.failures.map((one) => ({
    connectionName: one.connectionName,
    reason: one.reason,
    code: readFailureCode(one.code),
  })),
})

const writePriority = (priority: ConnectionCallPriority | undefined): Priority =>
  priority === 'interactive' ? Priority.INTERACTIVE : Priority.BACKGROUND

const writePreferredRow = (preferred: ConnectionCallPreferredRow | undefined) =>
  preferred === undefined
    ? {}
    : {
        preferredRow: {
          connectionId: preferred.connectionId,
          options: { ...preferred.options },
          isExclusive: preferred.isExclusive === true,
        },
      }

const openCancellation = (
  rpc: ConnectionCallRpc,
  pluginId: string,
  signal: AbortSignal | undefined,
) => {
  if (signal === undefined) {
    return { callId: '', close: () => undefined }
  }
  // An aborted fetch does not end the call in the core, so the abort cancels it by its id.
  const callId = crypto.randomUUID()
  const cancel = () => {
    rpc.cancelCall({ pluginId, callId }).catch(() => undefined)
  }
  signal.addEventListener('abort', cancel, { once: true })
  return { callId, close: () => signal.removeEventListener('abort', cancel) }
}

/** The calls carried out through a ConnectionCallService client. */
export const createConnectionCallClient = (rpc: ConnectionCallRpc): ConnectionCallClient => ({
  call: async function* (request, options) {
    const signal = options?.signal
    const cancellation = openCancellation(rpc, request.pluginId, signal)
    try {
      const stream = rpc.call(
        {
          ...request,
          callId: cancellation.callId,
          priority: writePriority(options?.priority),
          ...writePreferredRow(options?.preferred),
        },
        signal === undefined ? {} : { signal },
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
    } finally {
      cancellation.close()
    }
  },
  listRows: async (pluginId, operation) => {
    const { rows } = await rpc.listRows({ pluginId, operation })
    return rows.map((row) => ({
      connectionId: row.connectionId,
      connectionName: row.connectionName,
      options: { ...row.options },
      fields: row.fields.map(({ key, title, optionsFrom, isRequired }) => ({
        key,
        title,
        optionsFrom,
        isRequired,
      })),
    }))
  },
  listRowOptions: async (request) => {
    const { options } = await rpc.listRowOptions({ ...request, options: { ...request.options } })
    return options.map(({ value, title, group }) => ({ value, title, group }))
  },
})
