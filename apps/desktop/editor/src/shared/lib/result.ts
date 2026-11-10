/**
 * The answer to a call that can fail: what was returned, or why nothing was. A caller reads `ok`
 * to know which case it is in.
 */
export type Result<T, E = string> =
  { readonly ok: true; readonly value: T } | { readonly ok: false; readonly error: E }
