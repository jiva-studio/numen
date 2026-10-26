/**
 * What a call over the stencils, decks and presets failed on, in the words the
 * plugin's windows use. The codes of a note read and written are the shared
 * ones of `@numen/wire`; the codes of a stencil, a deck and a preset are this
 * plugin's.
 */
import type { ErrorCode as NoteProtoErrorCode } from '@numen/protocol'
import {
  ERRORS as NOTE_ERRORS,
  findErrorCode as findNoteErrorCode,
  formatErrorCodeMessage as formatNoteErrorCodeMessage,
  getErrorWords as getNoteErrorWords,
} from '@numen/wire'
import type { ErrorCode as NoteErrorCode } from '@numen/wire'
import { ErrorCode as ProtoErrorCode } from '../../gen/numen/flashcards/v1/errors_pb'

/** The plugin's own codes, which no other vault operation answers with. */
type OwnErrorCode = 'notAStencil' | 'notADeck' | 'deckTooLarge' | 'notAPreset'

/** The error code a call of this plugin answers with. */
export type ErrorCode = NoteErrorCode | OwnErrorCode

const OWN_CODE: Readonly<Partial<Record<ProtoErrorCode, OwnErrorCode>>> = {
  [ProtoErrorCode.NOT_A_STENCIL]: 'notAStencil',
  [ProtoErrorCode.NOT_A_DECK]: 'notADeck',
  [ProtoErrorCode.DECK_TOO_LARGE]: 'deckTooLarge',
  [ProtoErrorCode.NOT_A_PRESET]: 'notAPreset',
}

const OWN_WORDS: Readonly<Record<OwnErrorCode, string>> = {
  notAStencil: 'that note is not a stencil',
  notADeck: 'that note is not a deck',
  deckTooLarge: 'that deck is longer than this reads',
  notAPreset: 'that note is not a preset',
}

const isOwn = (code: ErrorCode): code is OwnErrorCode => code in OWN_WORDS

/** The shared code a note code of this plugin is numbered as. */
const toNoteCode = (code: ProtoErrorCode | undefined): NoteProtoErrorCode | undefined =>
  code as number | undefined as NoteProtoErrorCode | undefined

/** What one answer failed on, and nothing where it succeeded. */
export const findErrorCode = (from: { error?: ProtoErrorCode | undefined }): ErrorCode | null =>
  (from.error === undefined ? undefined : OWN_CODE[from.error]) ??
  findNoteErrorCode({ error: toNoteCode(from.error) })

/** Whether the file an answer is about had moved past what the caller read. */
export const isStaleAnswer = (from: { error?: ProtoErrorCode | undefined }): boolean =>
  from.error === ProtoErrorCode.STALE

/** What one error code says. */
export const getErrorWords = (code: ErrorCode): string =>
  isOwn(code) ? OWN_WORDS[code] : getNoteErrorWords(code)

/** What each error code says. */
export const ERRORS: Readonly<Record<ErrorCode, string>> = { ...NOTE_ERRORS, ...OWN_WORDS }

/** What one code of the schema says, and nothing where there was no error. */
export const formatErrorCodeMessage = (code: ProtoErrorCode | undefined): string => {
  const own = code === undefined ? undefined : OWN_CODE[code]
  return own ? OWN_WORDS[own] : formatNoteErrorCodeMessage(toNoteCode(code))
}
