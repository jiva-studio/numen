/** The answers of a core, written as a test puts them. */

export const asValue = <T>(value: T) => ({ ok: true as const, value })
export const asFailure = <E>(error: E) => ({ ok: false as const, error })
