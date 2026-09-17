/**
 * What a call that can fail answers with.
 *
 * A caller reads one field to know which case it is in, and the other field is
 * there only in that case. A bare value beside an optional error is two fields
 * to test and a third state nothing means: an empty value with no error.
 */

/** What went wrong, where a caller names no words of its own. */
export type Failure = string

/** The answer to a call that can fail: what came back, or why nothing did. */
export type Result<T, E = Failure> =
  { readonly ok: true; readonly value: T } | { readonly ok: false; readonly error: E }

/** The answer of a call that came back with something. */
export const asValue = <T>(value: T): Result<T, never> => ({ ok: true, value })

/** The answer of a call that did not. */
export const asFailure = <E>(error: E): Result<never, E> => ({ ok: false, error })
