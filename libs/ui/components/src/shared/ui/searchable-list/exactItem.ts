import type { ListItem } from './rows'

/** The item whose text is the query, ignoring case and the space around the query. */
export const findExactItem = (items: readonly ListItem[], query: string): ListItem | undefined => {
  const clean = query.trim().toLowerCase()
  if (!clean) return undefined
  return items.find((item) => item.text.toLowerCase() === clean)
}
