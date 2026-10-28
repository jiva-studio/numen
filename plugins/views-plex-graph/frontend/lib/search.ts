/**
 * Pure search matching helpers.
 */

export const SEARCH_DEBOUNCE_MS = 120

/** Checks whether any search result matches the query by title or note path. */
export function hasExactNoteMatch(
  results: readonly { readonly path: string; readonly title: string }[],
  query: string,
): boolean {
  const clean = query.trim().toLowerCase()
  if (!clean) return false
  return results.some(
    (item) =>
      item.title.toLowerCase() === clean ||
      item.path.toLowerCase() === clean ||
      item.path.toLowerCase() === `${clean}.md`,
  )
}
