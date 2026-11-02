/**
 * User-facing error messages. Each message is a clause, displayed after text
 * that identifies the failing item (a note title, a row), and a caller that
 * needs a full sentence composes one from it.
 */
import { Code, ConnectError } from '@connectrpc/connect'
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'

/** The message for failures with no more precise description. */
const UNEXPECTED = 'internal error'

/**
 * Messages by connection status code. Unknown is the code a lost connection
 * produces; a code that nothing here produces gets a message that states only
 * what is known; a call cancelled by the caller has an empty message.
 */
const TROUBLE: Record<Code, string> = {
  [Code.Canceled]: '',
  [Code.Unknown]: 'Numen is not responding; retrying',
  [Code.InvalidArgument]: 'invalid request',
  [Code.DeadlineExceeded]: 'request timed out',
  [Code.NotFound]: 'not found',
  [Code.AlreadyExists]: UNEXPECTED,
  [Code.PermissionDenied]: UNEXPECTED,
  [Code.ResourceExhausted]: UNEXPECTED,
  [Code.FailedPrecondition]: 'not possible right now',
  [Code.Aborted]: UNEXPECTED,
  [Code.OutOfRange]: UNEXPECTED,
  [Code.Unimplemented]: 'not supported by this build',
  [Code.Internal]: UNEXPECTED,
  [Code.Unavailable]: 'Numen is busy; try again',
  [Code.DataLoss]: UNEXPECTED,
  [Code.Unauthenticated]: UNEXPECTED,
}

/**
 * The codes the application raises only with a message of its own.
 *
 * Their messages are written for display and read as a clause: a vault that is
 * closing, a recording held by a run, a setting that is not valid JSON and the
 * byte where parsing stops. Not found is excluded because half of its messages
 * are `fs.ErrNotExist` with a path and a stat call prefixed.
 */
const CARRIES = new Set<Code>([
  Code.InvalidArgument,
  Code.FailedPrecondition,
  Code.Unavailable,
  Code.Unimplemented,
])

/**
 * The message for a caught error, shown to the user who awaited the result.
 */
export const formatErrorMessage = (error: unknown): string => {
  const fault = ConnectError.from(error)
  const carried = CARRIES.has(fault.code) ? fault.rawMessage.trim() : ''
  return carried || (TROUBLE[fault.code] ?? UNEXPECTED)
}

/**
 * The message for each error code in the schema.
 */
const ERROR_CODE_MESSAGES: Record<ProtoErrorCode, string> = {
  [ProtoErrorCode.UNSPECIFIED]: 'note unreadable',
  [ProtoErrorCode.MISSING]: 'note not in the vault',
  [ProtoErrorCode.NOT_A_NOTE]: 'not a note',
  [ProtoErrorCode.NOT_TEXT]: 'not a text file',
  [ProtoErrorCode.TOO_LARGE]: 'note too large',
  [ProtoErrorCode.BODY_UNWRITABLE]: 'text cannot start with frontmatter',
  [ProtoErrorCode.UNREADABLE]: 'frontmatter unreadable',
  [ProtoErrorCode.ALREADY_EXISTS]: 'a file with this name exists',
  [ProtoErrorCode.INVALID_NAME]: 'invalid note name',
  [ProtoErrorCode.STALE]: 'not saved: the note changed on disk',
}

/** The message for one error code; empty when there was no error. */
export const formatErrorCodeMessage = (errorCode: number | undefined): string =>
  errorCode === undefined ? '' : (ERROR_CODE_MESSAGES[errorCode as ProtoErrorCode] ?? UNEXPECTED)
