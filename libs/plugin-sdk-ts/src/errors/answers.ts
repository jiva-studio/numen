/**
 * What every client of the application shares: the file an answer came out of,
 * and what an error is called in the window's own words.
 *
 * A file is one value a caller carries about and never reads into. The schema
 * holds the parts; what a tab does with one is present it back unchanged, so
 * the parts stay here and the string goes everywhere else.
 */
import { Code, type ConnectError } from '@connectrpc/connect'
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import type { ErrorCode } from './errorCode'

/** The file an answer came out of, as the one string the window carries. */
export const formatFingerprint = (at?: {
  path: string
  size: bigint
  mtime: bigint
}): string | undefined => at && `${at.size} ${at.mtime} ${at.path}`

/** The two numbers first: a path holds spaces, and everything after them is it. */
export const parseFingerprint = (stat: string) => {
  // Stryker disable next-line StringLiteral: BigInt evaluates empty string and '0' both to 0n.
  const [size = '0', mtime = '0', ...rest] = stat.split(' ')
  return { path: rest.join(' '), size: BigInt(size), mtime: BigInt(mtime) }
}

/**
 * Maps each error code of the schema to the window's own message code.
 *
 * A file that changed since the caller read it has no message: it is a prompt
 * for the user, not a message, and the window reports it as `changed`. The map
 * is keyed by the schema, so a new error code must be given a message or an
 * explicit empty entry here before this compiles.
 */
export const ERROR_CODE: Record<number, ErrorCode | null> = {
  [ProtoErrorCode.UNSPECIFIED]: 'unreadable',
  [ProtoErrorCode.MISSING]: 'missing',
  [ProtoErrorCode.NOT_A_NOTE]: 'notANote',
  [ProtoErrorCode.NOT_TEXT]: 'notText',
  [ProtoErrorCode.TOO_LARGE]: 'tooLarge',
  [ProtoErrorCode.BODY_UNWRITABLE]: 'bodyUnwritable',
  [ProtoErrorCode.UNREADABLE]: 'unreadable',
  [ProtoErrorCode.ALREADY_EXISTS]: 'alreadyExists',
  [ProtoErrorCode.INVALID_NAME]: 'invalidName',
  [ProtoErrorCode.STALE]: null,
} satisfies Record<ProtoErrorCode, ErrorCode | null>

/** What one answer encountered as an error, and nothing where it succeeded. */
export const findErrorCode = (from: { error?: number | undefined }): ErrorCode | null =>
  (from.error !== undefined ? ERROR_CODE[from.error] : undefined) ?? null

/** Whether the file an answer is about had moved past what the caller read. */
export const isStaleAnswer = (from: { error?: number | undefined }): boolean =>
  from.error === ProtoErrorCode.STALE

/**
 * The address of a vault file query. The path is written in full, so a file in
 * a folder is one part of the address and the queried facet is the next.
 */
export const getAssetUrl = (path: string): string => `/assets/${encodeURIComponent(path)}`

/**
 * Which bytes an address refers to, as the address spells them. The path is
 * already part of the address, so this holds the remaining parts that identify
 * the file.
 */
export const getBytesQuery = (file: string): string => {
  const stat = parseFingerprint(file)
  return `size=${stat.size}&mtime=${stat.mtime}`
}

/** How often a document that is busy is waited out before it errors. */
export const MAX_BUSY_RETRIES = 3

/** How long the window waits before asking a busy document again. */
const BUSY_RETRY_DELAY_MS = 1000

/**
 * Retries a call while the application reports that the document is busy: a
 * document held by a client that is rendering from it is requested again after
 * a delay.
 */
export const retryWhileBusy = async <T>(ask: () => Promise<T>): Promise<T> => {
  for (let attempt = 0; ; attempt++) {
    try {
      return await ask()
    } catch (error) {
      if (Code.Unavailable !== (error as ConnectError).code || attempt >= MAX_BUSY_RETRIES)
        throw error
      await sleep(BUSY_RETRY_DELAY_MS)
    }
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
