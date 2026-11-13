import type { Result } from '../result'
import type { Disposable } from '../types'

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

// ConnectionPreferredRow is a connection among the rows of the plugin for the operation, asked before the
// other rows with `options` in place of the values of its row fields. A connection that is not among the
// rows is ignored.
export interface ConnectionPreferredRow {
  readonly connectionId: string
  readonly options: Readonly<Record<string, string>>
}

// ConnectionCallOptions are what a call carries besides its operation and payload. `preferred` is the
// row asked first; without it the rows are asked in their order.
export interface ConnectionCallOptions {
  readonly signal?: AbortSignal
  readonly priority?: ConnectionPriority
  readonly preferred?: ConnectionPreferredRow
}

// ConnectionRowField is a row field a connection type declares. `optionsFrom` is the operation that lists
// its values, empty for none. `isRequired` is a field a row must hold a value for.
export interface ConnectionRowField {
  readonly key: string
  readonly title: string
  readonly optionsFrom: string
  readonly isRequired: boolean
}

// ConnectionRow is one row of the plugin: its connection, the values of its row fields by key, and the
// row fields the type of the connection declares.
export interface ConnectionRow {
  readonly connectionId: string
  readonly connectionName: string
  readonly options: Readonly<Record<string, string>>
  readonly fields: readonly ConnectionRowField[]
}

// ConnectionRowOption is one value a row field can take. An empty title shows the value, and an empty
// group shows no heading.
export interface ConnectionRowOption {
  readonly value: string
  readonly title: string
  readonly group: string
}

// ConnectionRowOptionsRequest names a row field of a connection among the rows of an operation, with the
// values of the row fields so far.
export interface ConnectionRowOptionsRequest {
  readonly operation: string
  readonly connectionId: string
  readonly field: string
  readonly options: Readonly<Record<string, string>>
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
  // Lists the rows of the plugin for the operation in the order of use, without the rows whose connection
  // is missing or whose type is unavailable or does not serve the operation. A host that offers no choice
  // of rows lists none.
  listRows?(operation: string): Promise<readonly ConnectionRow[]>
  // Lists the values a row field of a connection among the rows of the operation can take, as its provider
  // lists them for the options of the request. It throws for a connection that is not among the rows and for a provider
  // that cannot list them.
  listRowOptions?(request: ConnectionRowOptionsRequest): Promise<readonly ConnectionRowOption[]>
  // Calls the handler after a change that can change the rows listRows lists: a connection, the
  // connections detected on this machine, or which plugins are enabled. A host that does not follow these
  // changes has no onChange.
  onChange?(handler: () => void): Disposable
}
