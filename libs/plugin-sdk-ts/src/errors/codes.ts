/** What the window says a command came back with, in the words each error code is given. */
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import { formatErrorCodeMessage } from './error'
import type { ErrorCode } from './errorCode'

/** The error code of the schema each of the window's codes is worded by. */
const PROTO_BY_CODE: Record<Exclude<ErrorCode, 'unreachable'>, ProtoErrorCode> = {
  missing: ProtoErrorCode.MISSING,
  notANote: ProtoErrorCode.NOT_A_NOTE,
  notText: ProtoErrorCode.NOT_TEXT,
  tooLarge: ProtoErrorCode.TOO_LARGE,
  bodyUnwritable: ProtoErrorCode.BODY_UNWRITABLE,
  unreadable: ProtoErrorCode.UNREADABLE,
  alreadyExists: ProtoErrorCode.ALREADY_EXISTS,
  invalidName: ProtoErrorCode.INVALID_NAME,
}

/** What is said when the vault did not answer at all. */
const UNREACHABLE = 'vault unreachable'

/** What one error code says. */
export const getErrorWords = (code: ErrorCode): string =>
  code === 'unreachable' ? UNREACHABLE : formatErrorCodeMessage(PROTO_BY_CODE[code])

/** What each error code says. */
export const ERRORS: Record<ErrorCode, string> = {
  ...(Object.fromEntries(
    Object.entries(PROTO_BY_CODE).map(([code, proto]) => [code, formatErrorCodeMessage(proto)]),
  ) as Record<Exclude<ErrorCode, 'unreachable'>, string>),
  unreachable: UNREACHABLE,
}

/** The code the window words for a code given as text, and `unreadable` for one it has no words for. */
export const getKnownErrorCode = (code: string): ErrorCode =>
  code in ERRORS ? (code as ErrorCode) : 'unreadable'
