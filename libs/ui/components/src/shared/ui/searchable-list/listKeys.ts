/** What a key does in a searchable list, as plain values. */
export type ListKey = 'close' | 'next' | 'previous' | 'first' | 'last' | 'choose'

const KEYS = new Map<string, ListKey>([
  ['Escape', 'close'],
  ['ArrowDown', 'next'],
  ['ArrowUp', 'previous'],
  ['Home', 'first'],
  ['End', 'last'],
  ['Enter', 'choose'],
])

/**
 * The list key a press is. Home and End belong to the caret while something is written. In the
 * multiple mode Control or Meta with K chooses, as Enter does.
 */
export function readListKey(
  event: KeyboardEvent,
  options: { readonly isMultiple: boolean; readonly isQueryEmpty: boolean },
): ListKey | undefined {
  if (options.isMultiple && (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
    return 'choose'
  }
  const key = KEYS.get(event.key)
  if ((key === 'first' || key === 'last') && !options.isQueryEmpty) return undefined
  return key
}
