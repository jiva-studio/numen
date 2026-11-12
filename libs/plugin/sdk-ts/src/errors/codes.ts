/** The message text for each error code a command can return. */
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import { formatErrorCodeMessage } from './error'
import type { RelationError } from '../workspace'
import type { ErrorCode } from './errorCode'

/** The error code of the schema each of the window's codes is worded by. */
const PROTO_BY_CODE: Record<Exclude<ErrorCode, 'unreachable'>, ProtoErrorCode> = {
  notFound: ProtoErrorCode.NOT_FOUND,
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

const getErrorWords = (code: ErrorCode): string =>
  code === 'unreachable' ? UNREACHABLE : formatErrorCodeMessage(PROTO_BY_CODE[code])

/** The message for a relationship operation that did not land. */
export const getRelationErrorWords = (error: RelationError): string => {
  switch (error) {
    case 'invalid':
      return 'these notes cannot be joined that way'
    case 'cycle':
      return 'a note cannot sit above itself in the hierarchy'
    case 'nameExhausted':
      return 'every untitled note name is taken'
    default:
      return getErrorWords(error)
  }
}
