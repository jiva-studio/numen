/**
 * What went wrong, in words a person reads. Both are clauses, read after
 * something that says which thing went wrong — a note's title, the row an
 * error is set beside — and a window that wants a sentence of one makes it
 * one.
 */
import { Code, ConnectError } from '@connectrpc/connect'
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'

/** What is said when nothing more precise can honestly be said. */
const UNEXPECTED = 'internal error'

/**
 * What a call that could not be answered says. Unknown is what a connection
 * that has gone arrives as, a code nothing here produces says what is true and
 * no more, and a call the window itself stopped says nothing.
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
 * The codes the application raises only with a sentence of its own.
 *
 * Everything under them is hand-written and reads as a clause: a vault that is
 * closing, a recording a run is holding, a setting that does not read as JSON
 * and the byte it stops at. Not found is out, because half of what it carries
 * is `fs.ErrNotExist` with a path and a stat call in front of it.
 */
const CARRIES = new Set<Code>([
  Code.InvalidArgument,
  Code.FailedPrecondition,
  Code.Unavailable,
  Code.Unimplemented,
])

/**
 * What a caught fault says to the person who was waiting for the answer.
 */
export const formatErrorMessage = (error: unknown): string => {
  const fault = ConnectError.from(error)
  const carried = CARRIES.has(fault.code) ? fault.rawMessage.trim() : ''
  return carried || (TROUBLE[fault.code] ?? UNEXPECTED)
}

/**
 * What each error code the schema carries says.
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

/** What one error code says, and nothing where there was no error. */
export const formatErrorCodeMessage = (errorCode: ProtoErrorCode | undefined): string =>
  errorCode === undefined ? '' : (ERROR_CODE_MESSAGES[errorCode] ?? UNEXPECTED)
