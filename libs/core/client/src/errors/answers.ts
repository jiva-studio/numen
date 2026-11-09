/**
 * What the answers of the core share: the file an answer came out of, and what an error is called
 * in the window's own words.
 *
 * A file is one value a caller carries about and never reads into. The schema holds the parts; what
 * a tab does with one is present it back unchanged, so the parts stay here and the string goes
 * everywhere else.
 */
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import type { ErrorCode } from './errorCode'

/** The two numbers first: a path holds spaces, and everything after them is it. */
export const parseFingerprint = (stat: string) => {
  // Stryker disable next-line StringLiteral: BigInt evaluates empty string and '0' both to 0n.
  const [size = '0', mtime = '0', ...rest] = stat.split(' ')
  return { path: rest.join(' '), size: BigInt(size), mtime: BigInt(mtime) }
}

/**
 * Maps each error code of the schema to the window's own message code.
 *
 * A file that changed since the caller read it has no message: it is a prompt for the user, not a
 * message, and the window reports it as `changed`. The map is keyed by the schema, so a new error
 * code must be given a message or an explicit empty entry here before this compiles.
 */
const ERROR_CODE: Record<number, ErrorCode | null> = {
  [ProtoErrorCode.UNSPECIFIED]: 'unreadable',
  [ProtoErrorCode.NOT_FOUND]: 'notFound',
  [ProtoErrorCode.NOT_A_NOTE]: 'notANote',
  [ProtoErrorCode.NOT_TEXT]: 'notText',
  [ProtoErrorCode.TOO_LARGE]: 'tooLarge',
  [ProtoErrorCode.BODY_UNWRITABLE]: 'bodyUnwritable',
  [ProtoErrorCode.UNREADABLE]: 'unreadable',
  [ProtoErrorCode.ALREADY_EXISTS]: 'alreadyExists',
  [ProtoErrorCode.INVALID_NAME]: 'invalidName',
  [ProtoErrorCode.CONFLICT]: null,
} satisfies Record<ProtoErrorCode, ErrorCode | null>

/** What one answer encountered as an error, and nothing where it succeeded. */
export const findErrorCode = (from: { error?: number | undefined }): ErrorCode | null =>
  // Stryker disable next-line ConditionalExpression: indexing ERROR_CODE by undefined yields undefined
  (from.error !== undefined ? ERROR_CODE[from.error] : undefined) ?? null

/** Whether the file an answer is about had moved past what the caller read. */
export const isStaleAnswer = (from: { error?: number | undefined }): boolean =>
  from.error === ProtoErrorCode.CONFLICT
