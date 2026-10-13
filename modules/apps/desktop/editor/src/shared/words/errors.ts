/** What the window says a command came back with, in the words `@numen/wire` gives each error code. */
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import { formatErrorCodeMessage } from '@numen/wire'
import type { ErrorCode } from '@/shared/errors'

/** The error code of the schema each of the window's codes is worded by. */
const PROTO_BY_CODE: Record<Exclude<ErrorCode, 'unreachable'>, ProtoErrorCode> = {
  missing: ProtoErrorCode.MISSING,
  notANote: ProtoErrorCode.NOT_A_NOTE,
  notText: ProtoErrorCode.NOT_TEXT,
  tooLarge: ProtoErrorCode.TOO_LARGE,
  bodyUnwritable: ProtoErrorCode.BODY_UNWRITABLE,
  unreadable: ProtoErrorCode.UNREADABLE,
  occupied: ProtoErrorCode.OCCUPIED,
  unnameable: ProtoErrorCode.UNNAMEABLE,
  notAStencil: ProtoErrorCode.NOT_A_STENCIL,
  notADeck: ProtoErrorCode.NOT_A_DECK,
  deckTooLarge: ProtoErrorCode.DECK_TOO_LARGE,
  notAPreset: ProtoErrorCode.NOT_A_PRESET,
}

/** What is said when the vault did not answer at all. */
const UNREACHABLE = 'the vault could not be reached'

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
