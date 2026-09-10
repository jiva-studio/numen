/**
 * Discriminated result pattern for fallible operations.
 *
 * A fallible operation returns either success carrying a value or failure
 * carrying an error discriminant.
 */
import type { ErrorCode } from './note'

export type Result<T, E = ErrorCode> =
  | { readonly ok: true; readonly value: T; readonly error?: never }
  | { readonly ok: false; readonly error: E; readonly value?: never }

/** A successful result carrying a value. */
export const ok = <T>(value: T): Result<T, never> => ({ ok: true, value })

/** A failure result carrying an error discriminant. */
export const err = <E = ErrorCode>(error: E): Result<never, E> => ({ ok: false, error })
