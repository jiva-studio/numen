/**
 * Pure search matching helpers.
 */

export const SEARCH_DEBOUNCE_MS = 120

/** The result whose title is the query, ignoring case and the space around the query. */
export function findExactNote<T extends { readonly title: string }>(
  results: readonly T[],
  query: string,
): T | undefined {
  const clean = query.trim().toLowerCase()
  if (!clean) return undefined
  return results.find((item) => item.title.toLowerCase() === clean)
}
