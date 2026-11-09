/**
 * What a searchable list shows, as plain values: which items a query keeps,
 * which row stands first, and where a group name is drawn.
 */
import { stepTo } from '@/shared/lib/dom/roving'

/** One entry of a searchable list. The identifier is the caller's, returned as given. */
export interface ListItem {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly group?: string
  readonly hasSeparator?: boolean
}

/** The key of the row that stands first, whatever it is made of. */
export const FIRST_KEY = ''

/** One row as drawn: the action row, or an item. */
export type ListRow =
  | { readonly key: string; readonly kind: 'action' }
  | { readonly key: string; readonly kind: 'item'; readonly item: ListItem }

/**
 * The items a query names. Every word has to be found in an item's text, detail
 * or group, in any case; no words name every item.
 */
export const filterItems = (items: readonly ListItem[], query: string): readonly ListItem[] => {
  // Stryker disable next-line MethodExpression,Regex: filter(Boolean) removes empty segments from whitespace runs
  const words = query.toLowerCase().split(/\s+/).filter(Boolean)
  // Stryker disable next-line ConditionalExpression: words.every on an empty array returns true for every item
  if (words.length === 0) return items
  return items.filter((item) => {
    const haystack = [item.text, item.detail ?? '', item.group ?? ''].join('\n').toLowerCase()
    return words.every((word) => haystack.includes(word))
  })
}

/** The item whose text is the query, ignoring case and the space around the query. */
export const findExactItem = (items: readonly ListItem[], query: string): ListItem | undefined => {
  const clean = query.trim().toLowerCase()
  if (!clean) return undefined
  return items.find((item) => item.text.toLowerCase() === clean)
}

/**
 * The rows for a query. With an action row, nothing typed shows no row at all;
 * otherwise the action row stands first and the items follow. An item whose text
 * is exactly the query takes the first row in place of the action, under the
 * same key, so the first row keeps its place as the results arrive.
 */
export const buildRows = (
  items: readonly ListItem[],
  query: string,
  hasAction: boolean,
): readonly ListRow[] => {
  const asItem = (item: ListItem, key = item.id): ListRow => ({ key, kind: 'item', item })
  if (!hasAction) return items.map((item) => asItem(item))
  if (!query.trim()) return []
  const exact = findExactItem(items, query)
  const first: ListRow = exact ? asItem(exact, FIRST_KEY) : { key: FIRST_KEY, kind: 'action' }
  return [first, ...items.filter((item) => item !== exact).map((item) => asItem(item))]
}

/** The group name drawn over each row: where an item's group differs from the row before it. */
export const getHeadings = (rows: readonly ListRow[]): readonly (string | undefined)[] => {
  let above: string | undefined
  return rows.map((row) => {
    const group = row.kind === 'item' ? row.item.group : undefined
    const heading = group !== above ? group : undefined
    above = group
    return heading
  })
}

/** The row a step from `from` lands on, wrapping at both ends; -1 where there is none. */
export const stepRow = (count: number, from: number, step: number): number =>
  stepTo(
    Array.from({ length: count }, (_, index) => index),
    { from, by: step },
    () => true,
  )
