import type { Result } from '../result'

// ConnectionPriority says whose action a call serves. A call that names none is background, so it waits
// behind the calls of a person's actions when its connection runs as many calls as it may.
type ConnectionPriority = 'interactive' | 'background'

// ConnectionErrorCode is why a call has no result. `canceled` is a call whose signal aborted; the other
// three are the errors of the Go client: no row can serve the operation, every row failed, and a
// connection found the request itself wrong.
export type ConnectionErrorCode = 'no_connection' | 'all_failed' | 'stopped' | 'canceled'

// ConnectionFailure is one row that did not answer: the name of its connection and the reason.
interface ConnectionFailure {
  readonly connectionName: string
  readonly reason: string
}

// ConnectionError is the error of a call. `failures` lists the rows that failed, in order, and is empty
// for `no_connection` and `canceled`.
export interface ConnectionError {
  readonly error: ConnectionErrorCode
  readonly message: string
  readonly failures: readonly ConnectionFailure[]
}

// ConnectionCallOptions are what a call carries besides its operation and payload.
export interface ConnectionCallOptions {
  readonly signal?: AbortSignal
  readonly priority?: ConnectionPriority
}

// ConnectionHost asks the core to carry out operations for the plugin. The core takes the connections
// from the rows of the setting of the plugin that is used for the operation; a request names no
// connection and a result holds no key. The core knows an operation by its name only: the payload and
// the frames are bytes of the contract the operation belongs to, and the modules under `contracts` give
// each contract a typed client. The calls and the errors are those of the Go client in
// `libs/plugin/sdk-go/connection`.
export interface ConnectionHost {
  // Asks the rows of the plugin, in order, to carry out the operation and yields its data frames. The
  // core goes to the next row only when a row fails before its first data frame. The generator returns
  // the done payload, or the error of the call with the frames before it already yielded.
  call(
    operation: string,
    payload: Uint8Array,
    options?: ConnectionCallOptions,
  ): AsyncGenerator<Uint8Array, Result<Uint8Array, ConnectionError>>
}
