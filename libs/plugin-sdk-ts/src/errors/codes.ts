/** The message text for each error code a command can return. */
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

/** The message shown when the vault did not respond at all. */
const UNREACHABLE = 'vault unreachable'

/** The message for one error code. */
export const getErrorWords = (code: ErrorCode): string =>
  code === 'unreachable' ? UNREACHABLE : formatErrorCodeMessage(PROTO_BY_CODE[code])

/** The message for each error code. */
export const ERRORS: Record<ErrorCode, string> = {
  ...(Object.fromEntries(
    Object.entries(PROTO_BY_CODE).map(([code, proto]) => [code, formatErrorCodeMessage(proto)]),
  ) as Record<Exclude<ErrorCode, 'unreachable'>, string>),
  unreachable: UNREACHABLE,
}

/** The code the window words for a code given as text, and `unreadable` for one it has no words for. */
export const getKnownErrorCode = (code: string): ErrorCode =>
  code in ERRORS ? (code as ErrorCode) : 'unreadable'
