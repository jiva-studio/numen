/**
 * What the answers of the core share: the file an answer came out of, and what an error is called
 * in the window's own words.
 *
 * A file is one value a caller carries about and never reads into. The schema holds the parts; what
 * a tab does with one is present it back unchanged, so the parts stay here and the string goes
 * everywhere else.
 */
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import type { ErrorCode } from '@numen/result'

/**
 * The two first parts are the file as `<size>:<version>:<hash> <mtime>`: a path holds spaces, and everything
 * after them is it.
 */
export const parseFingerprint = (stat: string) => {
  const [head = '', mtime, ...rest] = stat.split(' ')
  const [size, version, hash, ...extra] = head.split(':')
  if (
    size === undefined ||
    version === undefined ||
    hash === undefined ||
    extra.length > 0 ||
    mtime === undefined ||
    rest.join(' ') === ''
  )
    throw new Error(`not the fingerprint of a file: ${stat}`)
  return {
    path: rest.join(' '),
    size: BigInt(size),
    mtime: BigInt(mtime),
    hash,
    version: BigInt(version),
  }
}

/**
 * Maps each error code of the schema to the window's own message code.
 *
 * An answer with no error has no message, and neither has a file that changed since the caller read it: it is a prompt for the user, not a
 * message, and the window reports it as `changed`. The map is keyed by the schema, so a new error
 * code must be given a message or an explicit empty entry here before this compiles.
 */
const ERROR_CODE: Record<number, ErrorCode | null> = {
  [ProtoErrorCode.UNSPECIFIED]: null,
  [ProtoErrorCode.NOT_FOUND]: 'notFound',
  [ProtoErrorCode.NOT_A_NOTE]: 'notANote',
  [ProtoErrorCode.NOT_TEXT]: 'notText',
  [ProtoErrorCode.TOO_LARGE]: 'tooLarge',
  [ProtoErrorCode.BODY_UNWRITABLE]: 'bodyUnwritable',
  [ProtoErrorCode.UNREADABLE]: 'unreadable',
  [ProtoErrorCode.ALREADY_EXISTS]: 'alreadyExists',
  [ProtoErrorCode.INVALID_NAME]: 'invalidName',
  [ProtoErrorCode.CONFLICT]: null,
  [ProtoErrorCode.NO_VAULT]: 'unreachable',
  [ProtoErrorCode.OUTSIDE_VAULT]: 'unreadable',
  [ProtoErrorCode.LOCKED]: 'unreadable',
  [ProtoErrorCode.INTERNAL]: 'unreadable',
  [ProtoErrorCode.NOT_ATTEMPTED]: 'unreadable',
  [ProtoErrorCode.INVALID]: 'unreadable',
  [ProtoErrorCode.NOT_OWNER]: 'unreadable',
  [ProtoErrorCode.UNAVAILABLE]: 'unreachable',
  [ProtoErrorCode.NO_WINDOW]: 'unreachable',
  [ProtoErrorCode.TIMEOUT]: 'unreachable',
  [ProtoErrorCode.CANCELED]: 'unreadable',
  [ProtoErrorCode.NO_CONNECTION]: 'unreachable',
  [ProtoErrorCode.ALL_FAILED]: 'unreachable',
  [ProtoErrorCode.STOPPED]: 'unreadable',
  [ProtoErrorCode.VAULT_COPY]: 'unreadable',
  [ProtoErrorCode.VAULT_OVERLAPS]: 'unreadable',
  [ProtoErrorCode.LAST_VAULT]: 'unreadable',
  [ProtoErrorCode.VAULT_CURRENT]: 'unreadable',
  [ProtoErrorCode.NO_TRASH]: 'unreadable',
  [ProtoErrorCode.UNSAVED]: 'unreadable',
  [ProtoErrorCode.ACTIVATION_LIMIT_REACHED]: 'unreadable',
  [ProtoErrorCode.EXPIRED]: 'unreadable',
  [ProtoErrorCode.DISABLED]: 'unreadable',
  [ProtoErrorCode.PRODUCT_MISMATCH]: 'unreadable',
  [ProtoErrorCode.KEY_STORE_FAILED]: 'unreadable',
  [ProtoErrorCode.TYPE_UNAVAILABLE]: 'unreachable',
  [ProtoErrorCode.NO_MODEL]: 'unreachable',
  [ProtoErrorCode.RECIPE_MISMATCH]: 'unreadable',
  [ProtoErrorCode.LIMIT_EXCEEDED]: 'unreadable',
} satisfies Record<ProtoErrorCode, ErrorCode | null>

/** What one answer encountered as an error, and nothing where it succeeded. */
export const findErrorCode = (from: { error?: number | undefined }): ErrorCode | null =>
  // Stryker disable next-line ConditionalExpression: indexing ERROR_CODE by undefined yields undefined
  (from.error !== undefined ? ERROR_CODE[from.error] : undefined) ?? null

/** Whether the file an answer is about had moved past what the caller read. */
export const isStaleAnswer = (from: { error?: number | undefined }): boolean =>
  from.error === ProtoErrorCode.CONFLICT
