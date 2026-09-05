/**
 * What every client of the application shares: the file an answer came out of,
 * and what a refusal is called in the window's own words.
 *
 * A file is one value a caller carries about and never reads into. The schema
 * holds the parts; what a tab does with one is present it back unchanged, so
 * the parts stay here and the string goes everywhere else.
 */
import { Refusal } from '@numen/protocol'
import type { RefusalReason } from './core'

/** The file an answer came out of, as the one string the window carries. */
export const stamp = (at?: { path: string; size: bigint; mtime: bigint }): string | undefined =>
  at && `${at.size} ${at.mtime} ${at.path}`

/** The two numbers first: a path holds spaces, and everything after them is it. */
export const fingerprint = (at: string) => {
  const [size = '0', mtime = '0', ...rest] = at.split(' ')
  return { path: rest.join(' '), size: BigInt(size), mtime: BigInt(mtime) }
}

/**
 * What each refusal the schema carries is called in the window's own words.
 *
 * A file that moved past what the caller read is not among them: that one is a
 * question for the person and not a message, and the window carries it as
 * `changed`.
 */
export const REFUSAL: Partial<Record<Refusal, RefusalReason>> = {
  [Refusal.UNSPECIFIED]: 'unreadable',
  [Refusal.MISSING]: 'missing',
  [Refusal.NOT_A_NOTE]: 'notANote',
  [Refusal.NOT_TEXT]: 'notText',
  [Refusal.TOO_LARGE]: 'tooLarge',
  [Refusal.BODY_REFUSED]: 'bodyRefused',
  [Refusal.UNREADABLE]: 'unreadable',
  [Refusal.OCCUPIED]: 'occupied',
  [Refusal.UNNAMEABLE]: 'unnameable',
  [Refusal.NOT_A_STENCIL]: 'notAStencil',
  [Refusal.NOT_A_DECK]: 'notADeck',
  [Refusal.DECK_TOO_LARGE]: 'deckTooLarge',
  [Refusal.NOT_A_PRESET]: 'notAPreset',
}

/** What one answer was refused for, and nothing where it was not refused. */
export const refusalIn = (from: { refusal?: Refusal | undefined }): RefusalReason | null =>
  from.refusal === undefined ? null : (REFUSAL[from.refusal] ?? null)

/** Whether the file an answer is about had moved past what the caller read. */
export const staleIn = (from: { refusal?: Refusal | undefined }): boolean =>
  from.refusal === Refusal.STALE
