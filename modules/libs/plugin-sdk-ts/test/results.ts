/** The answers of a core, written as a test puts them. */

/** An answer that came back with something. */
export const asValue = <T>(value: T) => ({ ok: true as const, value })

/** An answer that did not. */
export const asFailure = <E>(error: E) => ({ ok: false as const, error })
