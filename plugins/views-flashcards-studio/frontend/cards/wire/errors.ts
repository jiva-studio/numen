/**
 * What a call over the card models, decks and presets failed on, in the words the
 * plugin's windows use. Every code of `numen.flashcards.v1.ErrorCode` is
 * translated here into the plugin's own `ErrorCode`.
 */
import { ErrorCode as ProtoErrorCode } from '../../gen/numen/flashcards/v1/errors_pb'

/** The error code a call of this plugin answers with. */
export type ErrorCode =
  | 'missing'
  | 'notANote'
  | 'notText'
  | 'tooLarge'
  | 'bodyUnwritable'
  | 'unreadable'
  | 'occupied'
  | 'unnameable'
  | 'unreachable'
  | 'notACardModel'
  | 'notADeck'
  | 'deckTooLarge'
  | 'notAPreset'

/** The code each schema code is, and nothing for `STALE`: the window asks the person about that one. */
const ERROR_CODE: Readonly<Record<ProtoErrorCode, ErrorCode | null>> = {
  [ProtoErrorCode.UNSPECIFIED]: 'unreadable',
  [ProtoErrorCode.MISSING]: 'missing',
  [ProtoErrorCode.NOT_A_NOTE]: 'notANote',
  [ProtoErrorCode.NOT_TEXT]: 'notText',
  [ProtoErrorCode.TOO_LARGE]: 'tooLarge',
  [ProtoErrorCode.BODY_UNWRITABLE]: 'bodyUnwritable',
  [ProtoErrorCode.UNREADABLE]: 'unreadable',
  [ProtoErrorCode.OCCUPIED]: 'occupied',
  [ProtoErrorCode.UNNAMEABLE]: 'unnameable',
  [ProtoErrorCode.NOT_A_CARD_MODEL]: 'notACardModel',
  [ProtoErrorCode.NOT_A_DECK]: 'notADeck',
  [ProtoErrorCode.DECK_TOO_LARGE]: 'deckTooLarge',
  [ProtoErrorCode.NOT_A_PRESET]: 'notAPreset',
  [ProtoErrorCode.STALE]: null,
}

/** What each error code says. */
export const ERRORS: Readonly<Record<ErrorCode, string>> = {
  missing: 'note not in the vault',
  notANote: 'not a note',
  notText: 'not a text file',
  tooLarge: 'note too large',
  bodyUnwritable: 'text cannot start with frontmatter',
  unreadable: 'frontmatter unreadable',
  occupied: 'a file with this name exists',
  unnameable: 'invalid note name',
  unreachable: 'vault unreachable',
  notACardModel: 'that note is not a card model',
  notADeck: 'that note is not a deck',
  deckTooLarge: 'that deck is longer than this reads',
  notAPreset: 'that note is not a preset',
}

/** What a schema code without a code of its own says. */
const SCHEMA_ONLY_WORDS: Readonly<Partial<Record<ProtoErrorCode, string>>> = {
  [ProtoErrorCode.UNSPECIFIED]: 'note unreadable',
  [ProtoErrorCode.STALE]: 'not saved: the note changed on disk',
}

/** What is said for a schema code this build does not know. */
const UNEXPECTED = 'internal error'

/** What one answer failed on, and nothing where it succeeded. */
export const findErrorCode = (from: { error?: ProtoErrorCode | undefined }): ErrorCode | null =>
  from.error === undefined ? null : (ERROR_CODE[from.error] ?? null)

/** Whether the file an answer is about had moved past what the caller read. */
export const isStaleAnswer = (from: { error?: ProtoErrorCode | undefined }): boolean =>
  from.error === ProtoErrorCode.STALE

/** What one error code says. */
export const getErrorWords = (code: ErrorCode): string => ERRORS[code]

/** What one code of the schema says, and nothing where there was no error. */
export const formatErrorCodeMessage = (code: ProtoErrorCode | undefined): string => {
  if (code === undefined) return ''
  const own = SCHEMA_ONLY_WORDS[code]
  if (own !== undefined) return own
  const pluginCode = ERROR_CODE[code]
  return pluginCode ? ERRORS[pluginCode] : UNEXPECTED
}
