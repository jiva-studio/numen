/**
 * User-facing error messages. Each message is a clause, displayed after text
 * that identifies the failing item (a note title, a row), and a caller that
 * needs a full sentence composes one from it.
 *
 * A connection fault is recognized by its shape and not by its class: every package that holds the
 * transport has its own copy of it, and an `instanceof` of this package's copy would miss the
 * faults of the others.
 */

/** The message for failures with no more precise description. */
const UNEXPECTED = 'internal error'

/** The status codes of a connection, as the transport numbers them. */
const STATUS = {
  Canceled: 1,
  Unknown: 2,
  InvalidArgument: 3,
  DeadlineExceeded: 4,
  NotFound: 5,
  AlreadyExists: 6,
  PermissionDenied: 7,
  ResourceExhausted: 8,
  FailedPrecondition: 9,
  Aborted: 10,
  OutOfRange: 11,
  Unimplemented: 12,
  Internal: 13,
  Unavailable: 14,
  DataLoss: 15,
  Unauthenticated: 16,
} as const

/**
 * Messages by connection status code. Unknown is the code a lost connection
 * produces; a code that nothing here produces gets a message that states only
 * what is known; a call cancelled by the caller has an empty message.
 */
const TROUBLE: Readonly<Partial<Record<number, string>>> = {
  [STATUS.Canceled]: '',
  [STATUS.Unknown]: 'Numen is not responding; retrying',
  [STATUS.InvalidArgument]: 'invalid request',
  [STATUS.DeadlineExceeded]: 'request timed out',
  [STATUS.NotFound]: 'not found',
  [STATUS.AlreadyExists]: UNEXPECTED,
  [STATUS.PermissionDenied]: UNEXPECTED,
  [STATUS.ResourceExhausted]: UNEXPECTED,
  [STATUS.FailedPrecondition]: 'not possible right now',
  [STATUS.Aborted]: UNEXPECTED,
  [STATUS.OutOfRange]: UNEXPECTED,
  [STATUS.Unimplemented]: 'not supported by this build',
  [STATUS.Internal]: UNEXPECTED,
  [STATUS.Unavailable]: 'Numen is busy; try again',
  [STATUS.DataLoss]: UNEXPECTED,
  [STATUS.Unauthenticated]: UNEXPECTED,
}

/**
 * The codes the application raises only with a message of its own.
 *
 * Their messages are written for display and read as a clause: a vault that is
 * closing, a recording held by a run, a setting that is not valid JSON and the
 * byte where parsing stops. Not found is excluded because half of its messages
 * are `fs.ErrNotExist` with a path and a stat call prefixed.
 */
const CARRIES: ReadonlySet<number> = new Set([
  STATUS.InvalidArgument,
  STATUS.FailedPrecondition,
  STATUS.Unavailable,
  STATUS.Unimplemented,
])

/** The name of a connection fault. */
const CONNECT_ERROR = 'ConnectError'

/** The name of an error whose message is shown as it is. Other errors carry backend or platform text and are shown by their code. */
const REPORTED = 'ReportedError'

/** The name of the error a cancelled fetch rejects with. */
const ABORT_ERROR = 'AbortError'

const hasName = (error: unknown, name: string): error is { readonly name: string } =>
  typeof error === 'object' && error !== null && 'name' in error && error.name === name

const isReportedError = (error: unknown): error is { readonly message: string } =>
  hasName(error, REPORTED) && 'message' in error && typeof error.message === 'string'

const isConnectFault = (
  error: unknown,
): error is { readonly code: number; readonly rawMessage: string } =>
  hasName(error, CONNECT_ERROR) &&
  'code' in error &&
  typeof error.code === 'number' &&
  'rawMessage' in error &&
  typeof error.rawMessage === 'string'

const getStatus = (error: unknown): { code: number; rawMessage: string } => {
  if (isConnectFault(error)) return error
  return { code: hasName(error, ABORT_ERROR) ? STATUS.Canceled : STATUS.Unknown, rawMessage: '' }
}

/**
 * An error whose message is shown as it is.
 * Other errors carry backend or platform text and are shown by their code.
 */
export class ReportedError extends Error {
  constructor(message: string) {
    super(message)
    this.name = REPORTED
  }
}

/**
 * The message for a caught error, shown to the user who awaited the result.
 */
export const formatErrorMessage = (error: unknown): string => {
  const reported = isReportedError(error) ? error.message.trim() : ''
  if (reported) return reported
  const { code, rawMessage } = getStatus(error)
  const carried = CARRIES.has(code) ? rawMessage.trim() : ''
  return carried || (TROUBLE[code] ?? UNEXPECTED)
}

/**
 * The message for each error code in the schema, by the number the schema gives it. A test of the
 * core client keeps this table complete against the schema.
 */
const ERROR_CODE_MESSAGES: Readonly<Partial<Record<number, string>>> = {
  0: 'note unreadable',
  1: 'note not in the vault',
  2: 'not a note',
  3: 'not a text file',
  4: 'note too large',
  5: 'text cannot start with frontmatter',
  6: 'frontmatter unreadable',
  7: 'a file with this name exists',
  8: 'invalid note name',
  9: 'not saved: the note changed on disk',
  10: 'no vault is open',
  11: 'the path is outside the vault',
  12: 'another program holds the file',
  13: 'the note could not be changed',
  14: 'the note was not changed',
  15: 'invalid request argument',
  16: 'not owned by this caller',
  17: 'service unavailable',
  18: 'no window is available',
  19: 'operation timed out',
  20: 'operation canceled',
  21: 'no connection available',
  22: 'every connection failed',
  23: 'request stopped by provider',
  24: 'vault is a copy of another registered vault',
  25: 'folder overlaps an existing vault',
  26: 'cannot remove the last vault',
  27: 'vault is currently open',
  28: 'no system trash is available',
  29: 'unsaved changes prevent this action',
  30: 'license activation limit reached',
  31: 'license key has expired',
  32: 'license key is disabled',
  33: 'license key is for another product',
  34: 'key store failed',
  35: 'connection type is unavailable',
  36: 'no search model is configured',
  37: 'search index recipe mismatch',
  38: 'limit exceeded',
}

/** The message for one error code; empty when there was no error. */
export const formatErrorCodeMessage = (errorCode: number | undefined): string =>
  errorCode === undefined ? '' : (ERROR_CODE_MESSAGES[errorCode] ?? UNEXPECTED)
