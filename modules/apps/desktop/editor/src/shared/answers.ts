/**
 * What every client of the application shares: the file an answer came out of,
 * and what an error is called in the window's own words.
 *
 * A file is one value a caller carries about and never reads into. The schema
 * holds the parts; what a tab does with one is present it back unchanged, so
 * the parts stay here and the string goes everywhere else.
 */
import { Code, type ConnectError } from '@connectrpc/connect'
import { Refusal as ProtoErrorCode } from '@numen/protocol'
import type { ErrorCode } from '@/shared/errors'

/** The file an answer came out of, as the one string the window carries. */
export const stamp = (at?: { path: string; size: bigint; mtime: bigint }): string | undefined =>
  at && `${at.size} ${at.mtime} ${at.path}`

/** The two numbers first: a path holds spaces, and everything after them is it. */
export const fingerprint = (at: string) => {
  const [size = '0', mtime = '0', ...rest] = at.split(' ')
  return { path: rest.join(' '), size: BigInt(size), mtime: BigInt(mtime) }
}

/**
 * What each error code the schema carries is called in the window's own words.
 *
 * A file that moved past what the caller read has no word: that one is a
 * question for the person and not a message, and the window carries it as
 * `changed`. Keyed by the schema, so an error code added to it has to be given a
 * word or that same silence here before this compiles.
 */
export const ERROR_CODE: Record<ProtoErrorCode, ErrorCode | null> = {
  [ProtoErrorCode.UNSPECIFIED]: 'unreadable',
  [ProtoErrorCode.MISSING]: 'missing',
  [ProtoErrorCode.NOT_A_NOTE]: 'notANote',
  [ProtoErrorCode.NOT_TEXT]: 'notText',
  [ProtoErrorCode.TOO_LARGE]: 'tooLarge',
  [ProtoErrorCode.BODY_REFUSED]: 'bodyRefused',
  [ProtoErrorCode.UNREADABLE]: 'unreadable',
  [ProtoErrorCode.OCCUPIED]: 'occupied',
  [ProtoErrorCode.UNNAMEABLE]: 'unnameable',
  [ProtoErrorCode.NOT_A_STENCIL]: 'notAStencil',
  [ProtoErrorCode.NOT_A_DECK]: 'notADeck',
  [ProtoErrorCode.DECK_TOO_LARGE]: 'deckTooLarge',
  [ProtoErrorCode.NOT_A_PRESET]: 'notAPreset',
  [ProtoErrorCode.STALE]: null,
}

/** What one answer encountered as an error, and nothing where it succeeded. */
export const errorIn = (from: { refusal?: ProtoErrorCode | undefined; error?: ProtoErrorCode | undefined }): ErrorCode | null => {
  const err = from.error ?? from.refusal
  return err === undefined ? null : (ERROR_CODE[err] ?? null)
}

/** Whether the file an answer is about had moved past what the caller read. */
export const staleIn = (from: { refusal?: ProtoErrorCode | undefined; error?: ProtoErrorCode | undefined }): boolean =>
  (from.error ?? from.refusal) === ProtoErrorCode.STALE

/**
 * Where a file of the vault is asked about. The path is written out whole, so a
 * file in a folder is one part of the address and the facet asked of it is the
 * next.
 */
export const asset = (path: string): string => `/assets/${encodeURIComponent(path)}`

/**
 * Which bytes an address is about, as the address writes them. The path is a
 * part of the address already, so what is written here is the rest of what says
 * which file it is.
 */
export const getBytesQuery = (seen: string): string => {
  const at = fingerprint(seen)
  return `size=${at.size}&mtime=${at.mtime}`
}

/** How often a document that is busy is waited out before it errors. */
const PATIENCE = 3

/** How long the window waits before asking a busy document again. */
const AGAIN = 1000

/**
 * What the application answered. A document held by whoever is drawing from it
 * says so, and is asked again after a wait.
 */
export const waiting = async <T>(ask: () => Promise<T>): Promise<T> => {
  for (let asked = 0; ; asked++) {
    try {
      return await ask()
    } catch (error) {
      if (Code.Unavailable !== (error as ConnectError).code || asked >= PATIENCE) throw error
      await sleep(AGAIN)
    }
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

