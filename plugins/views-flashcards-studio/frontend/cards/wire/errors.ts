/**
 * What a call over the card models, decks and presets failed on, in the words the
 * plugin's windows use. Every code of the API `ErrorCode` is
 * translated here into the plugin's own `ErrorCode`.
 */
import { ErrorCode as ProtoErrorCode } from '../../api/types'

/** The error code a call of this plugin answers with. */
export type ErrorCode =
  | 'missing'
  | 'notANote'
  | 'notText'
  | 'tooLarge'
  | 'bodyUnwritable'
  | 'unreadable'
  | 'alreadyExists'
  | 'invalidName'
  | 'unreachable'
  | 'notACardModel'
  | 'notADeck'
  | 'deckTooLarge'
  | 'notAPreset'

/** The code for each schema code, and null for `STALE`: the window asks the user about that one. */
const ERROR_CODE: Readonly<Record<ProtoErrorCode, ErrorCode | null>> = {
  [ProtoErrorCode.MISSING]: 'missing',
  [ProtoErrorCode.NOT_A_NOTE]: 'notANote',
  [ProtoErrorCode.NOT_TEXT]: 'notText',
  [ProtoErrorCode.TOO_LARGE]: 'tooLarge',
  [ProtoErrorCode.BODY_UNWRITABLE]: 'bodyUnwritable',
  [ProtoErrorCode.UNREADABLE]: 'unreadable',
  [ProtoErrorCode.ALREADY_EXISTS]: 'alreadyExists',
  [ProtoErrorCode.INVALID_NAME]: 'invalidName',
  [ProtoErrorCode.NOT_A_CARD_MODEL]: 'notACardModel',
  [ProtoErrorCode.NOT_A_DECK]: 'notADeck',
  [ProtoErrorCode.DECK_TOO_LARGE]: 'deckTooLarge',
  [ProtoErrorCode.NOT_A_PRESET]: 'notAPreset',
  [ProtoErrorCode.STALE]: null,
}

/** The message for each error code. */
export const ERRORS: Readonly<Record<ErrorCode, string>> = {
  missing: 'note not in the vault',
  notANote: 'not a note',
  notText: 'not a text file',
  tooLarge: 'note too large',
  bodyUnwritable: 'text cannot start with frontmatter',
  unreadable: 'frontmatter unreadable',
  alreadyExists: 'a file with this name exists',
  invalidName: 'invalid note name',
  unreachable: 'vault unreachable',
  notACardModel: 'that note is not a card model',
  notADeck: 'that note is not a deck',
  deckTooLarge: 'that deck is longer than this reads',
  notAPreset: 'that note is not a preset',
}

/** The message for a schema code that has no code of its own. */
const SCHEMA_ONLY_WORDS: Readonly<Partial<Record<ProtoErrorCode, string>>> = {
  [ProtoErrorCode.STALE]: 'not saved: the note changed on disk',
}

/** The message for a schema code that this build does not know. */
const UNEXPECTED = 'internal error'

/** What one answer failed on, and nothing where it succeeded. */
export const findErrorCode = (from: { error?: ProtoErrorCode | undefined }): ErrorCode | null =>
  from.error === undefined ? null : (ERROR_CODE[from.error] ?? null)

/** Whether the file an answer is about had moved past what the caller read. */
export const isStaleAnswer = (from: { error?: ProtoErrorCode | undefined }): boolean =>
  from.error === ProtoErrorCode.STALE

/** The message for one error code. */
export const getErrorWords = (code: ErrorCode): string => ERRORS[code]

/** The message for one schema code, and an empty string when there was no error. */
export const formatErrorCodeMessage = (code: ProtoErrorCode | undefined): string => {
  if (code === undefined) return ''
  const own = SCHEMA_ONLY_WORDS[code]
  if (own !== undefined) return own
  const pluginCode = ERROR_CODE[code]
  return pluginCode ? ERRORS[pluginCode] : UNEXPECTED
}
