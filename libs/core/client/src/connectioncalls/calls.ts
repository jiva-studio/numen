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
export type ConnectionCallRpc = Pick<Client<typeof ConnectionCallService>, 'call'>

/** Whose action a call serves. A call that names none is background. */
export type ConnectionCallPriority = 'interactive' | 'background'

/** What a call may carry besides its operation and payload. */
export interface ConnectionCallOptions {
  readonly signal?: AbortSignal | undefined
  readonly priority?: ConnectionCallPriority | undefined
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

/** The calls carried out through a ConnectionCallService client. */
export const createConnectionCallClient = (rpc: ConnectionCallRpc): ConnectionCallClient => ({
  call: async function* (request, options) {
    const stream = rpc.call(
      { ...request, priority: writePriority(options?.priority) },
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
})

/** The calls of the window the page is served from. */
export const connectionCalls = createConnectionCallClient(
  createClient(ConnectionCallService, transport),
)
